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

import { coursesApi } from '../../api/endpoints';
import type { Course } from '../../api/types';
import { notifyApiError, notifySuccess } from '../../lib/notify';

export function CoursesPage() {
    const qc = useQueryClient();
    const coursesQuery = useQuery({ queryKey: ['courses'], queryFn: () => coursesApi.list() });

    const [editOpen, { open: openEdit, close: closeEdit }] = useDisclosure(false);
    const [newOpen, { open: openNew, close: closeNew }] = useDisclosure(false);

    const form = useForm<{ id: string; name: string; code: string }>({
        initialValues: { id: '', name: '', code: '' },
        validate: {
            name: (v) => (v.trim() ? null : 'Название обязательно'),
            code: (v) => (v.trim() ? null : 'Код обязателен'),
        },
    });

    const createMut = useMutation({
        mutationFn: () =>
            coursesApi.create({ name: form.values.name.trim(), code: form.values.code.trim() }),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['courses'] });
            notifySuccess('Курс создан');
            closeNew();
        },
        onError: (e) => notifyApiError(e),
    });

    const updateMut = useMutation({
        mutationFn: () =>
            coursesApi.update(form.values.id, {
                name: form.values.name.trim(),
                code: form.values.code.trim(),
            }),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['courses'] });
            notifySuccess('Сохранено');
            closeEdit();
        },
        onError: (e) => notifyApiError(e),
    });

    const deleteMut = useMutation({
        mutationFn: (id: string) => coursesApi.delete(id),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['courses'] });
        },
        onError: (e) => notifyApiError(e),
    });

    const startEdit = (c: Course) => {
        form.setValues({ id: c.id, name: c.name, code: c.code });
        openEdit();
    };

    // Reset поля на close
    useEffect(() => {
        if (!editOpen && !newOpen) form.reset();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [editOpen, newOpen]);

    return (
        <Stack gap="md">
            <Group justify="space-between">
                <Title order={2}>Курсы</Title>
                <Button onClick={openNew}>Новый курс</Button>
            </Group>

            {coursesQuery.isLoading && <Loader />}
            {coursesQuery.data && (
                <Paper withBorder radius="md">
                    <Table striped highlightOnHover verticalSpacing="sm">
                        <Table.Thead>
                            <Table.Tr>
                                <Table.Th>Код</Table.Th>
                                <Table.Th>Название</Table.Th>
                                <Table.Th style={{ width: 120 }} />
                            </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody>
                            {coursesQuery.data.items.map((c) => (
                                <Table.Tr key={c.id}>
                                    <Table.Td>{c.code}</Table.Td>
                                    <Table.Td>{c.name}</Table.Td>
                                    <Table.Td>
                                        <Group gap="xs">
                                            <Button size="xs" variant="subtle" onClick={() => startEdit(c)}>
                                                Ред.
                                            </Button>
                                            <ActionIcon
                                                color="red"
                                                variant="subtle"
                                                onClick={() => {
                                                    if (confirm(`Удалить курс «${c.name}»?`))
                                                        deleteMut.mutate(c.id);
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

            <Modal opened={newOpen} onClose={closeNew} title="Новый курс" centered>
                <form onSubmit={form.onSubmit(() => createMut.mutate())}>
                    <Stack>
                        <TextInput label="Название" {...form.getInputProps('name')} />
                        <TextInput label="Код" {...form.getInputProps('code')} />
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

            <Modal opened={editOpen} onClose={closeEdit} title="Редактирование курса" centered>
                <form onSubmit={form.onSubmit(() => updateMut.mutate())}>
                    <Stack>
                        <TextInput label="Название" {...form.getInputProps('name')} />
                        <TextInput label="Код" {...form.getInputProps('code')} />
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
