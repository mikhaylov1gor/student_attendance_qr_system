import { Badge, Button, Group, Loader, Paper, Stack, Table, Text, Title } from '@mantine/core';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link } from 'react-router-dom';

import { policiesApi } from '../api/endpoints';
import { notifyApiError, notifySuccess } from '../lib/notify';

export function PoliciesListPage() {
    const qc = useQueryClient();
    const policiesQuery = useQuery({
        queryKey: ['policies'],
        queryFn: () => policiesApi.list(),
    });

    const setDefaultMut = useMutation({
        mutationFn: (id: string) => policiesApi.setDefault(id),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['policies'] });
            notifySuccess('Политика назначена политикой по умолчанию');
        },
        onError: (e) => notifyApiError(e),
    });

    return (
        <Stack gap="md">
            <Group justify="space-between">
                <Title order={2}>Политики безопасности</Title>
                <Button component={Link} to="/policies/new">
                    Создать
                </Button>
            </Group>

            {policiesQuery.isLoading && <Loader />}
            {policiesQuery.data && (
                <Paper withBorder radius="md">
                    <Table striped highlightOnHover verticalSpacing="sm">
                        <Table.Thead>
                            <Table.Tr>
                                <Table.Th>Название</Table.Th>
                                <Table.Th>QR TTL</Table.Th>
                                <Table.Th>Gео</Table.Th>
                                <Table.Th>Wi-Fi</Table.Th>
                                <Table.Th>Default</Table.Th>
                                <Table.Th />
                            </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody>
                            {policiesQuery.data.items.map((p) => (
                                <Table.Tr key={p.id}>
                                    <Table.Td>
                                        <Text component={Link} to={`/policies/${p.id}`} fw={500}>
                                            {p.name}
                                        </Text>
                                    </Table.Td>
                                    <Table.Td>
                                        {p.mechanisms.qr_ttl.enabled
                                            ? `${p.mechanisms.qr_ttl.ttl_seconds}s`
                                            : '—'}
                                    </Table.Td>
                                    <Table.Td>{p.mechanisms.geo.enabled ? 'вкл' : '—'}</Table.Td>
                                    <Table.Td>{p.mechanisms.wifi.enabled ? 'вкл' : '—'}</Table.Td>
                                    <Table.Td>
                                        {p.is_default ? <Badge color="green">default</Badge> : null}
                                    </Table.Td>
                                    <Table.Td>
                                        {!p.is_default && (
                                            <Button
                                                size="xs"
                                                variant="subtle"
                                                onClick={() => {
                                                    if (confirm(`Сделать «${p.name}» политикой по умолчанию?`))
                                                        setDefaultMut.mutate(p.id);
                                                }}
                                                loading={setDefaultMut.isPending}
                                            >
                                                Set default
                                            </Button>
                                        )}
                                    </Table.Td>
                                </Table.Tr>
                            ))}
                        </Table.Tbody>
                    </Table>
                </Paper>
            )}
        </Stack>
    );
}
