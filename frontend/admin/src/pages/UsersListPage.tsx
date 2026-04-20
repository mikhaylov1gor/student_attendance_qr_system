import {
    Badge,
    Button,
    Group,
    Loader,
    Paper,
    Select,
    Stack,
    Table,
    Text,
    TextInput,
    Title,
} from '@mantine/core';
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { Link } from 'react-router-dom';

import { usersApi } from '../api/endpoints';
import type { Role } from '../api/types';

const ROLE_LABELS: Record<Role, string> = {
    admin: 'Админ',
    teacher: 'Преподаватель',
    student: 'Студент',
};

export function UsersListPage() {
    const [q, setQ] = useState('');
    const [role, setRole] = useState<Role | ''>('');

    const usersQuery = useQuery({
        queryKey: ['users', { q, role }],
        queryFn: () =>
            usersApi.list({ q: q || undefined, role: role || undefined, limit: 100 }),
    });

    return (
        <Stack gap="md">
            <Group justify="space-between">
                <Title order={2}>Пользователи</Title>
                <Button component={Link} to="/users/new">
                    Создать
                </Button>
            </Group>

            <Paper withBorder p="md" radius="md">
                <Group>
                    <TextInput
                        label="Поиск по ФИО или email"
                        placeholder="Иванов"
                        value={q}
                        onChange={(e) => setQ(e.currentTarget.value)}
                    />
                    <Select
                        label="Роль"
                        placeholder="Любая"
                        clearable
                        data={[
                            { value: 'admin', label: ROLE_LABELS.admin },
                            { value: 'teacher', label: ROLE_LABELS.teacher },
                            { value: 'student', label: ROLE_LABELS.student },
                        ]}
                        value={role || null}
                        onChange={(v) => setRole((v as Role) ?? '')}
                    />
                </Group>
            </Paper>

            {usersQuery.isLoading && <Loader />}

            {usersQuery.data && (
                <Paper withBorder radius="md">
                    <Table striped highlightOnHover verticalSpacing="sm">
                        <Table.Thead>
                            <Table.Tr>
                                <Table.Th>ФИО</Table.Th>
                                <Table.Th>Email</Table.Th>
                                <Table.Th>Роль</Table.Th>
                                <Table.Th />
                            </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody>
                            {usersQuery.data.items.length === 0 && (
                                <Table.Tr>
                                    <Table.Td colSpan={4}>
                                        <Text c="dimmed" ta="center" py="md">
                                            Ничего не найдено.
                                        </Text>
                                    </Table.Td>
                                </Table.Tr>
                            )}
                            {usersQuery.data.items.map((u) => (
                                <Table.Tr key={u.id}>
                                    <Table.Td>
                                        <Text component={Link} to={`/users/${u.id}`} fw={500}>
                                            {u.full_name}
                                        </Text>
                                    </Table.Td>
                                    <Table.Td>{u.email}</Table.Td>
                                    <Table.Td>
                                        <Badge>{ROLE_LABELS[u.role]}</Badge>
                                    </Table.Td>
                                    <Table.Td>
                                        <Button component={Link} to={`/users/${u.id}`} size="xs" variant="subtle">
                                            Открыть
                                        </Button>
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
