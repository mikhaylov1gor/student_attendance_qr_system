import { Alert, Button, Group, Paper, Select, Stack, TextInput, Title } from '@mantine/core';
import { useForm } from '@mantine/form';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';

import { ApiError } from '../api/client';
import { groupsApi, usersApi } from '../api/endpoints';
import type { Role } from '../api/types';
import { TempPasswordModal } from '../components/TempPasswordModal';
import { notifyApiError } from '../lib/notify';

type FormValues = {
    email: string;
    role: Role;
    last: string;
    first: string;
    middle: string;
    current_group_id: string | null;
};

export function UserNewPage() {
    const navigate = useNavigate();
    const [tempPassword, setTempPassword] = useState<string | null>(null);

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
        validate: {
            email: (v) => (/^\S+@\S+\.\S+$/.test(v) ? null : 'Некорректный email'),
            last: (v) => (v.trim() ? null : 'Фамилия обязательна'),
            first: (v) => (v.trim() ? null : 'Имя обязательно'),
            current_group_id: (v, values) =>
                values.role === 'student' && !v ? 'Для студента нужна группа' : null,
        },
    });

    const createMut = useMutation({
        mutationFn: () =>
            usersApi.create({
                email: form.values.email.trim(),
                role: form.values.role,
                last: form.values.last.trim(),
                first: form.values.first.trim(),
                middle: form.values.middle.trim() || undefined,
                current_group_id:
                    form.values.role === 'student' ? form.values.current_group_id ?? undefined : undefined,
            }),
        onSuccess: (res) => {
            if (res.temp_password) {
                setTempPassword(res.temp_password);
            } else {
                navigate(`/users/${res.user.id}`);
            }
        },
        onError: (e) => {
            if (e instanceof ApiError && e.code === 'email_taken') {
                form.setFieldError('email', 'Email уже используется');
                return;
            }
            notifyApiError(e);
        },
    });

    return (
        <Stack gap="md" maw={640}>
            <Title order={2}>Новый пользователь</Title>

            {createMut.error instanceof ApiError && createMut.error.code !== 'email_taken' && (
                <Alert color="red">{createMut.error.message}</Alert>
            )}

            <Paper withBorder p="md" radius="md">
                <form onSubmit={form.onSubmit(() => createMut.mutate())}>
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
                        {form.values.role === 'student' && (
                            <Select
                                label="Группа"
                                data={
                                    groupsQuery.data?.items.map((g) => ({ value: g.id, label: g.name })) ?? []
                                }
                                searchable
                                {...form.getInputProps('current_group_id')}
                            />
                        )}
                        <Group justify="flex-end">
                            <Button variant="default" onClick={() => navigate('/users')}>
                                Отмена
                            </Button>
                            <Button type="submit" loading={createMut.isPending}>
                                Создать
                            </Button>
                        </Group>
                    </Stack>
                </form>
            </Paper>

            <TempPasswordModal
                opened={tempPassword !== null}
                tempPassword={tempPassword}
                title="Пользователь создан"
                onClose={() => {
                    const createdId = createMut.data?.user.id;
                    setTempPassword(null);
                    if (createdId) navigate(`/users/${createdId}`);
                }}
            />
        </Stack>
    );
}
