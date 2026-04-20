import { Alert, Badge, Button, Group, Loader, Paper, Stack, Text, Title } from '@mantine/core';
import { notifications } from '@mantine/notifications';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, useNavigate, useParams } from 'react-router-dom';

import { ApiError } from '../api/client';
import { reportsApi, sessionsApi } from '../api/endpoints';
import { formatDateTime } from '../lib/format';

export function SessionDetailsPage() {
    const { id } = useParams();
    const navigate = useNavigate();
    const qc = useQueryClient();

    const sessionQuery = useQuery({
        queryKey: ['session', id],
        queryFn: () => sessionsApi.get(id!),
        enabled: !!id,
    });

    const startMut = useMutation({
        mutationFn: () => sessionsApi.start(id!),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['sessions'] });
            qc.invalidateQueries({ queryKey: ['session', id] });
            navigate(`/sessions/${id}/live`);
        },
        onError: (e) => notifyApiError(e),
    });
    const closeMut = useMutation({
        mutationFn: () => sessionsApi.close(id!),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['sessions'] });
            qc.invalidateQueries({ queryKey: ['session', id] });
        },
        onError: (e) => notifyApiError(e),
    });
    const deleteMut = useMutation({
        mutationFn: () => sessionsApi.delete(id!),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['sessions'] });
            navigate('/sessions');
        },
        onError: (e) => notifyApiError(e),
    });

    const downloadReport = async () => {
        try {
            const blob = await reportsApi.downloadSessionXlsx(id!);
            const url = URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `attendance-${id}.xlsx`;
            a.click();
            URL.revokeObjectURL(url);
        } catch (e) {
            notifyApiError(e);
        }
    };

    if (sessionQuery.isLoading) return <Loader />;
    if (sessionQuery.error instanceof ApiError) {
        return <Alert color="red">Не удалось загрузить сессию: {sessionQuery.error.message}</Alert>;
    }
    const s = sessionQuery.data;
    if (!s) return null;

    return (
        <Stack gap="md">
            <Group justify="space-between">
                <Title order={2}>Сессия</Title>
                <Badge color={s.status === 'active' ? 'green' : s.status === 'draft' ? 'gray' : 'blue'}>
                    {s.status}
                </Badge>
            </Group>

            <Paper withBorder p="md" radius="md">
                <Stack gap="xs">
                    <Row label="Курс" value={s.course_id} />
                    <Row label="Аудитория" value={s.classroom_id ?? '— (онлайн)'} />
                    <Row label="Начало" value={formatDateTime(s.starts_at)} />
                    <Row label="Окончание" value={formatDateTime(s.ends_at)} />
                    <Row label="QR TTL" value={`${s.qr_ttl_seconds} сек`} />
                    <Row label="Групп" value={s.group_ids.length.toString()} />
                </Stack>
            </Paper>

            <Group>
                {s.status === 'draft' && (
                    <>
                        <Button onClick={() => startMut.mutate()} loading={startMut.isPending}>
                            Start
                        </Button>
                        <Button color="red" variant="subtle" onClick={() => deleteMut.mutate()} loading={deleteMut.isPending}>
                            Удалить
                        </Button>
                    </>
                )}
                {s.status === 'active' && (
                    <>
                        <Button component={Link} to={`/sessions/${s.id}/live`}>
                            Открыть live
                        </Button>
                        <Button color="red" variant="light" onClick={() => closeMut.mutate()} loading={closeMut.isPending}>
                            Закрыть
                        </Button>
                    </>
                )}
                {s.status === 'closed' && (
                    <Button onClick={downloadReport}>Скачать xlsx-отчёт</Button>
                )}
            </Group>
        </Stack>
    );
}

function Row({ label, value }: { label: string; value: string }) {
    return (
        <Group justify="space-between">
            <Text c="dimmed">{label}</Text>
            <Text>{value}</Text>
        </Group>
    );
}

function notifyApiError(e: unknown) {
    const msg = e instanceof ApiError ? `${e.code}: ${e.message}` : String(e);
    notifications.show({ color: 'red', title: 'Ошибка', message: msg });
}
