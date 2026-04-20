import {
    Alert,
    Button,
    Group,
    MultiSelect,
    NumberInput,
    Paper,
    Select,
    Stack,
    Title,
} from '@mantine/core';
import { DateTimePicker } from '@mantine/dates';
import { useForm } from '@mantine/form';
import { notifications } from '@mantine/notifications';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';

import { ApiError } from '../api/client';
import { catalogApi, sessionsApi } from '../api/endpoints';
import type { CreateSessionRequest } from '../api/types';

// Mantine 9 DateTimePicker отдаёт строку (`YYYY-MM-DD HH:mm:ss`), а не Date.
type FormValues = {
    course_id: string | null;
    classroom_id: string | null;
    starts_at: string | null;
    ends_at: string | null;
    group_ids: string[];
    qr_ttl_seconds: number | null;
};

export function SessionCreatePage() {
    const navigate = useNavigate();
    const qc = useQueryClient();

    const coursesQuery = useQuery({
        queryKey: ['courses'],
        queryFn: () => catalogApi.courses.list(),
        staleTime: 60_000,
    });
    const classroomsQuery = useQuery({
        queryKey: ['classrooms'],
        queryFn: () => catalogApi.classrooms.list(),
        staleTime: 60_000,
    });

    const form = useForm<FormValues>({
        initialValues: {
            course_id: null,
            classroom_id: null,
            starts_at: null,
            ends_at: null,
            group_ids: [],
            qr_ttl_seconds: null,
        },
        validate: {
            course_id: (v) => (v ? null : 'Выберите курс'),
            starts_at: (v) => (v ? null : 'Укажите начало'),
            ends_at: (v, values) => {
                if (!v) return 'Укажите окончание';
                if (values.starts_at && new Date(v) <= new Date(values.starts_at))
                    return 'Окончание должно быть позже начала';
                return null;
            },
            group_ids: (v) => (v.length > 0 ? null : 'Выберите хотя бы одну группу'),
            qr_ttl_seconds: (v) => {
                if (v == null) return null;
                if (v < 3 || v > 120) return 'TTL должен быть в диапазоне 3–120';
                return null;
            },
        },
    });

    // Группы подгружаются в зависимости от выбранного курса (через streams).
    const streamsQuery = useQuery({
        queryKey: ['streams', form.values.course_id],
        queryFn: () => catalogApi.streams.listForCourse(form.values.course_id!),
        enabled: !!form.values.course_id,
    });
    const groupsQuery = useQuery({
        queryKey: ['groups'],
        queryFn: () => catalogApi.groups.list(),
        staleTime: 60_000,
    });

    // Пересечение: группы, которые входят хотя бы в один stream выбранного курса.
    const availableGroups = (() => {
        if (!streamsQuery.data || !groupsQuery.data) return [];
        const allowedIds = new Set<string>();
        streamsQuery.data.items.forEach((s) => s.group_ids.forEach((g) => allowedIds.add(g)));
        return groupsQuery.data.items
            .filter((g) => allowedIds.has(g.id))
            .map((g) => ({ value: g.id, label: g.name }));
    })();

    const createMut = useMutation({
        mutationFn: (body: CreateSessionRequest) => sessionsApi.create(body),
        onSuccess: (s) => {
            qc.invalidateQueries({ queryKey: ['sessions'] });
            notifications.show({ color: 'green', message: 'Черновик сессии создан' });
            navigate(`/sessions/${s.id}`);
        },
    });

    const onSubmit = form.onSubmit((v) => {
        createMut.mutate({
            course_id: v.course_id!,
            classroom_id: v.classroom_id || undefined,
            starts_at: new Date(v.starts_at!).toISOString(),
            ends_at: new Date(v.ends_at!).toISOString(),
            group_ids: v.group_ids,
            qr_ttl_seconds: v.qr_ttl_seconds ?? undefined,
        });
    });

    return (
        <Stack gap="md" maw={720}>
            <Title order={2}>Новая сессия</Title>

            {createMut.error instanceof ApiError && (
                <Alert color="red">
                    {createMut.error.code === 'groups_not_in_course_streams'
                        ? 'Одна из выбранных групп не принадлежит потокам этого курса.'
                        : createMut.error.message}
                </Alert>
            )}

            <Paper withBorder p="md" radius="md">
                <form onSubmit={onSubmit}>
                    <Stack>
                        <Select
                            label="Курс"
                            placeholder="Выберите"
                            searchable
                            data={
                                coursesQuery.data?.items.map((c) => ({
                                    value: c.id,
                                    label: `${c.code} · ${c.name}`,
                                })) ?? []
                            }
                            {...form.getInputProps('course_id')}
                        />
                        <Select
                            label="Аудитория"
                            placeholder="Опционально"
                            searchable
                            clearable
                            data={
                                classroomsQuery.data?.items.map((c) => ({
                                    value: c.id,
                                    label: `${c.building}, ${c.room_number}`,
                                })) ?? []
                            }
                            {...form.getInputProps('classroom_id')}
                        />
                        <Group grow>
                            <DateTimePicker label="Начало" {...form.getInputProps('starts_at')} />
                            <DateTimePicker label="Окончание" {...form.getInputProps('ends_at')} />
                        </Group>
                        <MultiSelect
                            label="Группы"
                            description={
                                form.values.course_id
                                    ? 'Доступны группы, входящие в потоки выбранного курса.'
                                    : 'Сначала выберите курс.'
                            }
                            searchable
                            data={availableGroups}
                            disabled={!form.values.course_id}
                            {...form.getInputProps('group_ids')}
                        />
                        <NumberInput
                            label="QR TTL (секунд)"
                            description="Опционально. Если пусто — используется значение из политики."
                            min={3}
                            max={120}
                            {...form.getInputProps('qr_ttl_seconds')}
                        />
                        <Group justify="flex-end">
                            <Button variant="default" onClick={() => navigate('/sessions')}>
                                Отмена
                            </Button>
                            <Button type="submit" loading={createMut.isPending}>
                                Создать черновик
                            </Button>
                        </Group>
                    </Stack>
                </form>
            </Paper>
        </Stack>
    );
}
