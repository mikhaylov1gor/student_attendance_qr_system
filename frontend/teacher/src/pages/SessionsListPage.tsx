import {
    ActionIcon,
    Alert,
    Badge,
    Button,
    Group,
    Loader,
    Paper,
    Select,
    Stack,
    Table,
    Text,
    Title,
    Tooltip,
} from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';

import { ApiError } from '../api/client';
import { catalogApi, sessionsApi } from '../api/endpoints';
import type { Session, SessionStatus } from '../api/types';
import { useAuthStore } from '../auth/store';
import { formatDateTime } from '../lib/format';

const STATUS_COLORS: Record<SessionStatus, string> = {
    draft: 'gray',
    active: 'green',
    closed: 'blue',
};

export function SessionsListPage() {
    const navigate = useNavigate();
    const qc = useQueryClient();
    const principal = useAuthStore((s) => s.principal);

    const [statusFilter, setStatusFilter] = useState<SessionStatus | ''>('');
    const [courseFilter, setCourseFilter] = useState<string>('');

    // Teacher видит только свои сессии. Admin — все.
    const teacherFilter = principal?.role === 'teacher' ? principal.id : undefined;

    const sessionsQuery = useQuery({
        queryKey: ['sessions', { teacher: teacherFilter, status: statusFilter, course: courseFilter }],
        queryFn: () =>
            sessionsApi.list({
                teacher_id: teacherFilter,
                status: statusFilter || undefined,
                course_id: courseFilter || undefined,
                limit: 100,
            }),
    });

    const coursesQuery = useQuery({
        queryKey: ['courses'],
        queryFn: () => catalogApi.courses.list(),
        staleTime: 60_000,
    });

    const coursesByID = useMemo(() => {
        const map = new Map<string, string>();
        coursesQuery.data?.items.forEach((c) => map.set(c.id, `${c.code} · ${c.name}`));
        return map;
    }, [coursesQuery.data]);

    const startMut = useMutation({
        mutationFn: (id: string) => sessionsApi.start(id),
        onSuccess: (s) => {
            qc.invalidateQueries({ queryKey: ['sessions'] });
            notifications.show({ color: 'green', message: 'Сессия запущена' });
            navigate(`/sessions/${s.id}/live`);
        },
        onError: (e) => notifyApiError(e),
    });

    const closeMut = useMutation({
        mutationFn: (id: string) => sessionsApi.close(id),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['sessions'] });
            notifications.show({ color: 'blue', message: 'Сессия закрыта' });
        },
        onError: (e) => notifyApiError(e),
    });

    return (
        <Stack gap="md">
            <Group justify="space-between">
                <Title order={2}>Сессии</Title>
                <Button component={Link} to="/sessions/new">
                    Новая сессия
                </Button>
            </Group>

            <Paper withBorder p="md" radius="md">
                <Group>
                    <Select
                        label="Статус"
                        placeholder="Все"
                        clearable
                        data={[
                            { value: 'draft', label: 'Черновик' },
                            { value: 'active', label: 'Активна' },
                            { value: 'closed', label: 'Закрыта' },
                        ]}
                        value={statusFilter || null}
                        onChange={(v) => setStatusFilter((v as SessionStatus) ?? '')}
                    />
                    <Select
                        label="Курс"
                        placeholder="Все"
                        searchable
                        clearable
                        data={
                            coursesQuery.data?.items.map((c) => ({
                                value: c.id,
                                label: `${c.code} · ${c.name}`,
                            })) ?? []
                        }
                        value={courseFilter || null}
                        onChange={(v) => setCourseFilter(v ?? '')}
                    />
                </Group>
            </Paper>

            {sessionsQuery.isLoading && <Loader />}

            {sessionsQuery.error instanceof ApiError && (
                <Alert color="red">Не удалось загрузить сессии: {sessionsQuery.error.message}</Alert>
            )}

            {sessionsQuery.data && (
                <Paper withBorder radius="md">
                    <Table striped highlightOnHover verticalSpacing="sm">
                        <Table.Thead>
                            <Table.Tr>
                                <Table.Th>Курс</Table.Th>
                                <Table.Th>Начало</Table.Th>
                                <Table.Th>Окончание</Table.Th>
                                <Table.Th>Статус</Table.Th>
                                <Table.Th>QR TTL</Table.Th>
                                <Table.Th style={{ width: 220 }}>Действия</Table.Th>
                            </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody>
                            {sessionsQuery.data.items.length === 0 && (
                                <Table.Tr>
                                    <Table.Td colSpan={6}>
                                        <Text c="dimmed" ta="center" py="md">
                                            Сессий пока нет.
                                        </Text>
                                    </Table.Td>
                                </Table.Tr>
                            )}
                            {sessionsQuery.data.items.map((s) => (
                                <SessionRow
                                    key={s.id}
                                    s={s}
                                    courseLabel={coursesByID.get(s.course_id) ?? s.course_id}
                                    onStart={() => startMut.mutate(s.id)}
                                    onClose={() => closeMut.mutate(s.id)}
                                    busy={startMut.isPending || closeMut.isPending}
                                />
                            ))}
                        </Table.Tbody>
                    </Table>
                </Paper>
            )}
        </Stack>
    );
}

function SessionRow({
    s,
    courseLabel,
    onStart,
    onClose,
    busy,
}: {
    s: Session;
    courseLabel: string;
    onStart: () => void;
    onClose: () => void;
    busy: boolean;
}) {
    return (
        <Table.Tr>
            <Table.Td>
                <Text component={Link} to={`/sessions/${s.id}`} fw={500}>
                    {courseLabel}
                </Text>
            </Table.Td>
            <Table.Td>{formatDateTime(s.starts_at)}</Table.Td>
            <Table.Td>{formatDateTime(s.ends_at)}</Table.Td>
            <Table.Td>
                <Badge color={STATUS_COLORS[s.status]}>{s.status}</Badge>
            </Table.Td>
            <Table.Td>{s.qr_ttl_seconds}s</Table.Td>
            <Table.Td>
                <Group gap="xs">
                    {s.status === 'draft' && (
                        <Tooltip label="Запустить и открыть live-экран">
                            <Button size="xs" onClick={onStart} loading={busy}>
                                Start
                            </Button>
                        </Tooltip>
                    )}
                    {s.status === 'active' && (
                        <>
                            <Button
                                component={Link}
                                to={`/sessions/${s.id}/live`}
                                size="xs"
                                variant="light"
                            >
                                Live
                            </Button>
                            <ActionIcon
                                variant="subtle"
                                color="red"
                                onClick={onClose}
                                loading={busy}
                                title="Закрыть сессию"
                            >
                                ✕
                            </ActionIcon>
                        </>
                    )}
                    {s.status === 'closed' && (
                        <Button
                            component={Link}
                            to={`/sessions/${s.id}`}
                            size="xs"
                            variant="subtle"
                        >
                            Детали
                        </Button>
                    )}
                </Group>
            </Table.Td>
        </Table.Tr>
    );
}

function notifyApiError(e: unknown) {
    const msg = e instanceof ApiError ? `${e.code}: ${e.message}` : String(e);
    notifications.show({ color: 'red', title: 'Ошибка', message: msg });
}
