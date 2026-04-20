import {
    ActionIcon,
    Alert,
    Badge,
    Button,
    Center,
    Drawer,
    Group,
    Paper,
    Stack,
    Table,
    Text,
    TextInput,
    Title,
    Tooltip,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { notifications } from '@mantine/notifications';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useEffect, useMemo, useState } from 'react';
import QRCode from 'react-qr-code';
import { useNavigate, useParams } from 'react-router-dom';

import { ApiError } from '../api/client';
import { attendanceApi, sessionsApi } from '../api/endpoints';
import type { AttendanceStatus, WsAttendanceMessage, WsQRMessage } from '../api/types';
import { useFullscreen } from '../hooks/useFullscreen';
import { useTeacherSocket } from '../hooks/useTeacherSocket';
import { formatTime } from '../lib/format';

type Row = {
    attendance_id: string;
    student_id: string;
    submitted_at: string;
    preliminary_status: AttendanceStatus;
    final_status?: 'accepted' | 'rejected';
    checks: WsAttendanceMessage['checks'];
};

export function SessionLivePage() {
    const { id } = useParams();
    const navigate = useNavigate();
    const { ref: fullscreenRef, toggle: toggleFullscreen, fullscreen } = useFullscreen();

    const [qr, setQR] = useState<WsQRMessage | null>(null);
    const [rows, setRows] = useState<Row[]>([]);
    const [secondsLeft, setSecondsLeft] = useState<number>(0);

    const sessionQuery = useQuery({
        queryKey: ['session', id],
        queryFn: () => sessionsApi.get(id!),
        enabled: !!id,
    });

    const { status: wsStatus, messages } = useTeacherSocket(id);

    // Diff-подход: в messages может нарасти произвольная история, поэтому
    // берём только те индексы, которых мы ещё не обработали.
    const [processedCount, setProcessedCount] = useState(0);
    useEffect(() => {
        if (messages.length <= processedCount) return;
        const fresh = messages.slice(processedCount);
        setProcessedCount(messages.length);
        for (const m of fresh) {
            if (m.type === 'qr_token') {
                setQR(m);
            } else if (m.type === 'attendance') {
                const row: Row = {
                    attendance_id: m.attendance_id,
                    student_id: m.student_id,
                    submitted_at: m.submitted_at,
                    preliminary_status: m.preliminary_status,
                    checks: m.checks,
                };
                setRows((prev) => [row, ...prev]);
            } else if (m.type === 'attendance_resolved') {
                setRows((prev) =>
                    prev.map((r) =>
                        r.attendance_id === m.attendance_id
                            ? { ...r, final_status: m.final_status }
                            : r,
                    ),
                );
            }
        }
    }, [messages, processedCount]);

    // Countdown до expires_at.
    useEffect(() => {
        if (!qr) return;
        const update = () => {
            const diff = new Date(qr.expires_at).getTime() - Date.now();
            setSecondsLeft(Math.max(0, Math.ceil(diff / 1000)));
        };
        update();
        const timer = setInterval(update, 250);
        return () => clearInterval(timer);
    }, [qr]);

    const closeMut = useMutation({
        mutationFn: () => sessionsApi.close(id!),
        onSuccess: () => navigate('/sessions'),
        onError: (e) => notifyApiError(e),
    });

    const session = sessionQuery.data;

    return (
        <div ref={fullscreenRef} style={{ minHeight: '100vh', background: 'var(--mantine-color-body)' }}>
            <Group justify="space-between" p="md">
                <Group>
                    <Button variant="subtle" onClick={() => navigate('/sessions')}>
                        ← К списку
                    </Button>
                    {session && (
                        <Text c="dimmed">
                            {formatTime(session.starts_at)} — {formatTime(session.ends_at)}
                        </Text>
                    )}
                    <Badge color={wsStatus === 'open' ? 'green' : wsStatus === 'connecting' ? 'yellow' : 'red'}>
                        WS: {wsStatus}
                    </Badge>
                </Group>
                <Group>
                    <Tooltip label="Полный экран">
                        <Button variant="light" onClick={() => toggleFullscreen()}>
                            {fullscreen ? 'Выйти из fullscreen' : 'На весь экран'}
                        </Button>
                    </Tooltip>
                    <Button
                        color="red"
                        variant="light"
                        onClick={() => closeMut.mutate()}
                        loading={closeMut.isPending}
                    >
                        Закрыть сессию
                    </Button>
                </Group>
            </Group>

            <div style={{ display: 'grid', gridTemplateColumns: '1fr 520px', gap: 24, padding: '0 24px 24px' }}>
                <QRPanel qr={qr} secondsLeft={secondsLeft} />
                <AttendanceTable rows={rows} />
            </div>
        </div>
    );
}

function QRPanel({ qr, secondsLeft }: { qr: WsQRMessage | null; secondsLeft: number }) {
    return (
        <Paper withBorder radius="md" p="xl">
            <Stack align="center" gap="md">
                {qr ? (
                    <>
                        <div style={{ width: '80vh', maxWidth: '100%', aspectRatio: '1 / 1' }}>
                            <QRCode
                                value={qr.token}
                                size={2048}
                                style={{ width: '100%', height: '100%' }}
                                level="M"
                            />
                        </div>
                        <Group gap="xl">
                            <Text size="xl">
                                Counter: <b>{qr.counter}</b>
                            </Text>
                            <Text size="xl" c={secondsLeft <= 2 ? 'red' : undefined}>
                                Обновится через: <b>{secondsLeft}s</b>
                            </Text>
                        </Group>
                    </>
                ) : (
                    <Center mih={400}>
                        <Text c="dimmed">Ожидаем первый QR-токен из канала…</Text>
                    </Center>
                )}
            </Stack>
        </Paper>
    );
}

function AttendanceTable({ rows }: { rows: Row[] }) {
    const total = rows.length;
    const accepted = useMemo(
        () => rows.filter((r) => (r.final_status ?? r.preliminary_status) === 'accepted').length,
        [rows],
    );
    const review = useMemo(
        () => rows.filter((r) => !r.final_status && r.preliminary_status === 'needs_review').length,
        [rows],
    );

    return (
        <Paper withBorder radius="md" p="md">
            <Stack gap="xs">
                <Group justify="space-between">
                    <Title order={4}>Отметки · {total}</Title>
                    <Group gap="xs">
                        <Badge color="green" variant="light">
                            accepted: {accepted}
                        </Badge>
                        <Badge color="yellow" variant="light">
                            нужно проверить: {review}
                        </Badge>
                    </Group>
                </Group>
                <Table verticalSpacing="xs" striped highlightOnHover>
                    <Table.Thead>
                        <Table.Tr>
                            <Table.Th>Время</Table.Th>
                            <Table.Th>Студент</Table.Th>
                            <Table.Th>Статус</Table.Th>
                            <Table.Th />
                        </Table.Tr>
                    </Table.Thead>
                    <Table.Tbody>
                        {rows.map((r) => (
                            <AttendanceRow key={r.attendance_id} r={r} />
                        ))}
                        {rows.length === 0 && (
                            <Table.Tr>
                                <Table.Td colSpan={4}>
                                    <Text c="dimmed" ta="center" py="md">
                                        Пока никто не отметился.
                                    </Text>
                                </Table.Td>
                            </Table.Tr>
                        )}
                    </Table.Tbody>
                </Table>
            </Stack>
        </Paper>
    );
}

function AttendanceRow({ r }: { r: Row }) {
    const effective = r.final_status ?? r.preliminary_status;
    const color =
        effective === 'accepted' ? 'green' : effective === 'rejected' ? 'red' : 'yellow';

    const [opened, { open, close }] = useDisclosure(false);
    const [notes, setNotes] = useState('');

    const resolveMut = useMutation({
        mutationFn: (final: 'accepted' | 'rejected') =>
            attendanceApi.resolve(r.attendance_id, { final_status: final, notes: notes || undefined }),
        onSuccess: () => {
            notifications.show({ color: 'green', message: 'Отметка обновлена' });
            // WS пришлёт attendance_resolved — таблица подтянется сама.
            close();
            setNotes('');
        },
        onError: (e) => notifyApiError(e),
    });

    const needsReview = !r.final_status && r.preliminary_status === 'needs_review';

    return (
        <>
            <Table.Tr>
                <Table.Td>{formatTime(r.submitted_at)}</Table.Td>
                <Table.Td style={{ fontFamily: 'ui-monospace, monospace' }}>
                    {r.student_id.slice(0, 8)}…
                </Table.Td>
                <Table.Td>
                    <Badge color={color}>
                        {r.final_status ? `${r.final_status} (override)` : r.preliminary_status}
                    </Badge>
                </Table.Td>
                <Table.Td>
                    {needsReview && (
                        <Group gap="xs">
                            <ActionIcon
                                color="green"
                                variant="light"
                                onClick={() => resolveMut.mutate('accepted')}
                                loading={resolveMut.isPending}
                                title="Принять"
                            >
                                ✓
                            </ActionIcon>
                            <ActionIcon
                                color="red"
                                variant="light"
                                onClick={() => resolveMut.mutate('rejected')}
                                loading={resolveMut.isPending}
                                title="Отклонить"
                            >
                                ✗
                            </ActionIcon>
                            <ActionIcon variant="subtle" onClick={open} title="Добавить комментарий">
                                ⋯
                            </ActionIcon>
                        </Group>
                    )}
                </Table.Td>
            </Table.Tr>

            <Drawer opened={opened} onClose={close} title="Override с комментарием" position="right">
                <Stack>
                    {r.checks && r.checks.length > 0 && (
                        <Alert variant="light">
                            <Stack gap={4}>
                                {r.checks.map((c) => (
                                    <Text key={c.mechanism} size="sm">
                                        <b>{c.mechanism}:</b> {c.status}
                                    </Text>
                                ))}
                            </Stack>
                        </Alert>
                    )}
                    <TextInput
                        label="Комментарий"
                        placeholder="Например: подошёл лично, подтвердил присутствие"
                        value={notes}
                        onChange={(e) => setNotes(e.currentTarget.value)}
                    />
                    <Group justify="flex-end">
                        <Button color="red" variant="light" onClick={() => resolveMut.mutate('rejected')}>
                            Отклонить
                        </Button>
                        <Button color="green" onClick={() => resolveMut.mutate('accepted')}>
                            Принять
                        </Button>
                    </Group>
                </Stack>
            </Drawer>
        </>
    );
}

function notifyApiError(e: unknown) {
    const msg =
        e instanceof ApiError
            ? e.code === 'not_resolvable'
                ? 'Запись уже resolved другим действием.'
                : e.code === 'forbidden'
                  ? 'Нет прав на эту сессию.'
                  : `${e.code}: ${e.message}`
            : String(e);
    notifications.show({ color: 'red', title: 'Ошибка', message: msg });
}
