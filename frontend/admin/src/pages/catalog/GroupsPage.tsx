import {
    ActionIcon,
    Button,
    Group,
    Loader,
    Modal,
    Paper,
    Stack,
    Table,
    TextInput,
    Title,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { useDisclosure } from '@mantine/hooks';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { groupsApi } from '../../api/endpoints';
import type { Group as GroupT } from '../../api/types';
import { notifyApiError, notifySuccess } from '../../lib/notify';

export function GroupsPage() {
    const qc = useQueryClient();
    const groupsQuery = useQuery({ queryKey: ['groups'], queryFn: () => groupsApi.list() });

    const [editOpen, { open: openEdit, close: closeEdit }] = useDisclosure(false);
    const [newOpen, { open: openNew, close: closeNew }] = useDisclosure(false);

    const form = useForm<{ id: string; name: string }>({
        initialValues: { id: '', name: '' },
        validate: {
            name: (v) => (v.trim() ? null : 'Название обязательно'),
        },
    });

    const createMut = useMutation({
        mutationFn: () => groupsApi.create({ name: form.values.name.trim() }),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['groups'] });
            notifySuccess('Группа создана');
            closeNew();
        },
        onError: (e) => notifyApiError(e),
    });

    const updateMut = useMutation({
        mutationFn: () => groupsApi.update(form.values.id, { name: form.values.name.trim() }),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['groups'] });
            notifySuccess('Сохранено');
            closeEdit();
        },
        onError: (e) => notifyApiError(e),
    });

    const deleteMut = useMutation({
        mutationFn: (id: string) => groupsApi.delete(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: ['groups'] }),
        onError: (e) => notifyApiError(e),
    });

    const startEdit = (g: GroupT) => {
        form.setValues({ id: g.id, name: g.name });
        openEdit();
    };

    useEffect(() => {
        if (!editOpen && !newOpen) form.reset();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [editOpen, newOpen]);

    return (
        <Stack gap="md">
            <Group justify="space-between">
                <Title order={2}>Группы</Title>
                <Button onClick={openNew}>Новая группа</Button>
            </Group>

            {groupsQuery.isLoading && <Loader />}
            {groupsQuery.data && (
                <Paper withBorder radius="md">
                    <Table striped highlightOnHover verticalSpacing="sm">
                        <Table.Thead>
                            <Table.Tr>
                                <Table.Th>Название</Table.Th>
                                <Table.Th style={{ width: 120 }} />
                            </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody>
                            {groupsQuery.data.items.map((g) => (
                                <Table.Tr key={g.id}>
                                    <Table.Td>{g.name}</Table.Td>
                                    <Table.Td>
                                        <Group gap="xs">
                                            <Button size="xs" variant="subtle" onClick={() => startEdit(g)}>
                                                Ред.
                                            </Button>
                                            <ActionIcon
                                                color="red"
                                                variant="subtle"
                                                onClick={() => {
                                                    if (confirm(`Удалить группу «${g.name}»?`))
                                                        deleteMut.mutate(g.id);
                                                }}
                                            >
                                                ✕
                                            </ActionIcon>
                                        </Group>
                                    </Table.Td>
                                </Table.Tr>
                            ))}
                        </Table.Tbody>
                    </Table>
                </Paper>
            )}

            <Modal opened={newOpen} onClose={closeNew} title="Новая группа" centered>
                <form onSubmit={form.onSubmit(() => createMut.mutate())}>
                    <Stack>
                        <TextInput label="Название (например, БПИ-241)" {...form.getInputProps('name')} />
                        <Group justify="flex-end">
                            <Button variant="default" onClick={closeNew}>
                                Отмена
                            </Button>
                            <Button type="submit" loading={createMut.isPending}>
                                Создать
                            </Button>
                        </Group>
                    </Stack>
                </form>
            </Modal>

            <Modal opened={editOpen} onClose={closeEdit} title="Редактирование группы" centered>
                <form onSubmit={form.onSubmit(() => updateMut.mutate())}>
                    <Stack>
                        <TextInput label="Название" {...form.getInputProps('name')} />
                        <Group justify="flex-end">
                            <Button variant="default" onClick={closeEdit}>
                                Отмена
                            </Button>
                            <Button type="submit" loading={updateMut.isPending}>
                                Сохранить
                            </Button>
                        </Group>
                    </Stack>
                </form>
            </Modal>
        </Stack>
    );
}
