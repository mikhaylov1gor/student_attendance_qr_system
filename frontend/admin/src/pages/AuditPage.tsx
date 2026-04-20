import {
    Alert,
    Badge,
    Button,
    Code,
    Group,
    Loader,
    Paper,
    Stack,
    Table,
    Text,
    TextInput,
    Title,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useState } from 'react';

import { auditApi } from '../api/endpoints';
import type { AuditEntry } from '../api/types';
import { formatDateTime, shortHash } from '../lib/format';
import { notifyApiError } from '../lib/notify';

export function AuditPage() {
    const qc = useQueryClient();
    const [action, setAction] = useState('');
    const [entityId, setEntityId] = useState('');

    const entriesQuery = useQuery({
        queryKey: ['audit', { action, entityId }],
        queryFn: () =>
            auditApi.list({
                action: action || undefined,
                entity_id: entityId || undefined,
                limit: 100,
            }),
    });

    const verifyMut = useMutation({
        mutationFn: () => auditApi.verify(),
        onError: (e) => notifyApiError(e, 'Не удалось проверить цепочку'),
    });

    return (
        <Stack gap="md">
            <Group justify="space-between">
                <Title order={2}>Журнал аудита</Title>
                <Button
                    onClick={() => {
                        verifyMut.mutate();
                        qc.invalidateQueries({ queryKey: ['audit'] });
                    }}
                    loading={verifyMut.isPending}
                    color={verifyMut.data?.ok === false ? 'red' : undefined}
                    variant={verifyMut.data?.ok === false ? 'filled' : 'filled'}
                >
                    Verify chain
                </Button>
            </Group>

            {verifyMut.data && <VerifyResultBanner result={verifyMut.data} />}

            <Paper withBorder p="md" radius="md">
                <Group>
                    <TextInput
                        label="Action"
                        placeholder="например, session_started"
                        value={action}
                        onChange={(e) => setAction(e.currentTarget.value)}
                    />
                    <TextInput
                        label="Entity ID"
                        placeholder="uuid или внешний id"
                        value={entityId}
                        onChange={(e) => setEntityId(e.currentTarget.value)}
                    />
                </Group>
            </Paper>

            {entriesQuery.isLoading && <Loader />}

            {entriesQuery.data && (
                <Paper withBorder radius="md">
                    <Table striped highlightOnHover verticalSpacing="sm">
                        <Table.Thead>
                            <Table.Tr>
                                <Table.Th>ID</Table.Th>
                                <Table.Th>Время</Table.Th>
                                <Table.Th>Action</Table.Th>
                                <Table.Th>Entity</Table.Th>
                                <Table.Th>Actor</Table.Th>
                                <Table.Th>Hash</Table.Th>
                                <Table.Th />
                            </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody>
                            {entriesQuery.data.items.map((e, idx) => {
                                const firstBroken = verifyMut.data?.first_broken_id;
                                const broken = firstBroken != null && e.id >= firstBroken;
                                return (
                                    <AuditRow
                                        key={e.id}
                                        e={e}
                                        broken={broken}
                                        showArrow={idx < entriesQuery.data.items.length - 1}
                                    />
                                );
                            })}
                        </Table.Tbody>
                    </Table>
                </Paper>
            )}
        </Stack>
    );
}

function VerifyResultBanner({
    result,
}: {
    result: { ok: boolean; total_entries: number; first_broken_id?: number; broken_reason?: string };
}) {
    if (result.ok) {
        return (
            <Alert color="green" title="Цепочка целостна ✓">
                Проверено {result.total_entries} записей. Хэш-цепочка SHA-256 consistent от genesis
                до последней записи.
            </Alert>
        );
    }
    return (
        <Alert color="red" title="ВНИМАНИЕ: цепочка нарушена ✗">
            <Stack gap={4}>
                <Text>Проверено: {result.total_entries}</Text>
                <Text>
                    Первая сломанная запись: <b>#{result.first_broken_id}</b>
                </Text>
                {result.broken_reason && (
                    <Text>
                        Причина: <Code>{result.broken_reason}</Code>
                    </Text>
                )}
            </Stack>
        </Alert>
    );
}

function AuditRow({ e, broken, showArrow }: { e: AuditEntry; broken: boolean; showArrow: boolean }) {
    const [opened, { toggle }] = useDisclosure(false);
    return (
        <>
            <Table.Tr style={{ background: broken ? 'var(--mantine-color-red-0)' : undefined }}>
                <Table.Td>
                    <Text fw={500}>#{e.id}</Text>
                    {showArrow && (
                        <Text size="xs" c="dimmed">
                            ↑ prev_hash
                        </Text>
                    )}
                </Table.Td>
                <Table.Td>{formatDateTime(e.occurred_at)}</Table.Td>
                <Table.Td>
                    <Badge variant="light">{e.action}</Badge>
                </Table.Td>
                <Table.Td>
                    <Text size="sm">{e.entity_type}</Text>
                    <Text size="xs" c="dimmed" style={{ fontFamily: 'ui-monospace, monospace' }}>
                        {e.entity_id.slice(0, 8)}…
                    </Text>
                </Table.Td>
                <Table.Td>
                    <Text size="sm">{e.actor_role || '—'}</Text>
                </Table.Td>
                <Table.Td style={{ fontFamily: 'ui-monospace, monospace' }}>
                    <Text size="xs">{shortHash(e.record_hash)}</Text>
                </Table.Td>
                <Table.Td>
                    <Button size="xs" variant="subtle" onClick={toggle}>
                        {opened ? 'Скрыть' : 'Payload'}
                    </Button>
                </Table.Td>
            </Table.Tr>
            {opened && (
                <Table.Tr>
                    <Table.Td colSpan={7} style={{ padding: 0, borderTop: 'none' }}>
                        <Stack gap={4} p="md">
                            <Text size="xs" c="dimmed">
                                prev_hash: <Code>{e.prev_hash}</Code>
                            </Text>
                            <Text size="xs" c="dimmed">
                                record_hash: <Code>{e.record_hash}</Code>
                            </Text>
                            {e.payload && (
                                <Code block style={{ maxHeight: 240, overflow: 'auto' }}>
                                    {JSON.stringify(e.payload, null, 2)}
                                </Code>
                            )}
                        </Stack>
                    </Table.Td>
                </Table.Tr>
            )}
        </>
    );
}
