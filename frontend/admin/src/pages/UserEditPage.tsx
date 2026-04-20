import {
    Alert,
    Badge,
    Button,
    Group,
    Loader,
    Paper,
    Select,
    Stack,
    TextInput,
    Title,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import { ApiError } from '../api/client';
import { groupsApi, usersApi } from '../api/endpoints';
import type { Role } from '../api/types';
import { TempPasswordModal } from '../components/TempPasswordModal';
import { notifyApiError, notifySuccess } from '../lib/notify';

type FormValues = {
    email: string;
    role: Role;
    last: string;
    first: string;
    middle: string;
    current_group_id: string | null;
};

function splitFullName(full: string): { last: string; first: string; middle: string } {
    const parts = full.trim().split(/\s+/);
    return {
        last: parts[0] ?? '',
        first: parts[1] ?? '',
        middle: parts.slice(2).join(' '),
    };
}

export function UserEditPage() {
    const { id } = useParams();
    const navigate = useNavigate();
    const qc = useQueryClient();
    const [tempPassword, setTempPassword] = useState<string | null>(null);

    const userQuery = useQuery({
        queryKey: ['user', id],
        queryFn: () => usersApi.get(id!),
        enabled: !!id,
    });
    const groupsQuery = useQuery({
        queryKey: ['groups'],
        queryFn: () => groupsApi.list(),
        staleTime: 60_000,
    });

    const form = useForm<FormValues>({
        initialValues: {
            email: '',
            role: 'student',
            last: '',
            first: '',
            middle: '',
            current_group_id: null,
        },
    });

    useEffect(() => {
        if (userQuery.data) {
            const name = splitFullName(userQuery.data.full_name);
            form.setValues({
                email: userQuery.data.email,
                role: userQuery.data.role,
                last: name.last,
                first: name.first,
                middle: name.middle,
                current_group_id: userQuery.data.current_group_id ?? null,
            });
            form.resetDirty();
        }
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [userQuery.data]);

    const updateMut = useMutation({
        mutationFn: () => {
            const original = userQuery.data!;
            const origName = splitFullName(original.full_name);
            const body: Record<string, unknown> = {};
            if (form.values.email !== original.email) body.email = form.values.email.trim();
            if (form.values.role !== original.role) body.role = form.values.role;
            if (form.values.last !== origName.last) body.last = form.values.last.trim();
            if (form.values.first !== origName.first) body.first = form.values.first.trim();
            if (form.values.middle !== origName.middle) body.middle = form.values.middle.trim();

            // clear_group / current_group_id
            const currentGroup = original.current_group_id ?? null;
            if (form.values.current_group_id !== currentGroup) {
                if (form.values.current_group_id) {
                    body.current_group_id = form.values.current_group_id;
                } else {
                    body.clear_group = true;
                }
            }
            return usersApi.update(id!, body);
        },
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['user', id] });
            qc.invalidateQueries({ queryKey: ['users'] });
            notifySuccess('Сохранено');
        },
        onError: (e) => {
            if (e instanceof ApiError && e.code === 'email_taken') {
                form.setFieldError('email', 'Email уже используется');
                return;
            }
            notifyApiError(e);
        },
    });

    const resetPasswordMut = useMutation({
        mutationFn: () => usersApi.resetPassword(id!),
        onSuccess: (res) => setTempPassword(res.temp_password),
        onError: (e) => notifyApiError(e),
    });

    const deleteMut = useMutation({
        mutationFn: () => usersApi.delete(id!),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['users'] });
            navigate('/users');
        },
        onError: (e) => notifyApiError(e),
    });

    if (userQuery.isLoading) return <Loader />;
    if (userQuery.error instanceof ApiError) {
        return <Alert color="red">Ошибка загрузки: {userQuery.error.message}</Alert>;
    }
    const u = userQuery.data;
    if (!u) return null;

    return (
        <Stack gap="md" maw={640}>
            <Group justify="space-between">
                <Title order={2}>{u.full_name}</Title>
                <Badge>{u.role}</Badge>
            </Group>

            <Paper withBorder p="md" radius="md">
                <form onSubmit={form.onSubmit(() => updateMut.mutate())}>
                    <Stack>
                        <TextInput label="Email" {...form.getInputProps('email')} />
                        <Select
                            label="Роль"
                            data={[
                                { value: 'student', label: 'Студент' },
                                { value: 'teacher', label: 'Преподаватель' },
                                { value: 'admin', label: 'Админ' },
                            ]}
                            {...form.getInputProps('role')}
                            allowDeselect={false}
                        />
                        <Group grow>
                            <TextInput label="Фамилия" {...form.getInputProps('last')} />
                            <TextInput label="Имя" {...form.getInputProps('first')} />
                            <TextInput label="Отчество" {...form.getInputProps('middle')} />
                        </Group>
                        <Select
                            label="Группа"
                            description={form.values.role === 'student' ? undefined : 'Для не-студента группа игнорируется.'}
                            clearable
                            searchable
                            data={groupsQuery.data?.items.map((g) => ({ value: g.id, label: g.name })) ?? []}
                            {...form.getInputProps('current_group_id')}
                        />
                        <Group justify="space-between" mt="sm">
                            <Group>
                                <Button
                                    color="red"
                                    variant="light"
                                    onClick={() => {
                                        if (confirm('Удалить пользователя?')) deleteMut.mutate();
                                    }}
                                    loading={deleteMut.isPending}
                                >
                                    Удалить
                                </Button>
                                <Button
                                    variant="light"
                                    onClick={() => resetPasswordMut.mutate()}
                                    loading={resetPasswordMut.isPending}
                                >
                                    Сбросить пароль
                                </Button>
                            </Group>
                            <Button type="submit" loading={updateMut.isPending}>
                                Сохранить
                            </Button>
                        </Group>
                    </Stack>
                </form>
            </Paper>

            <TempPasswordModal
                opened={tempPassword !== null}
                tempPassword={tempPassword}
                title="Новый временный пароль"
                onClose={() => setTempPassword(null)}
            />
        </Stack>
    );
}
