import {
    ActionIcon,
    Alert,
    Button,
    Group,
    Loader,
    Modal,
    MultiSelect,
    Paper,
    Select,
    Stack,
    Table,
    TextInput,
    Title,
} from '@mantine/core';
import { useForm } from '@mantine/form';
import { useDisclosure } from '@mantine/hooks';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect, useMemo, useState } from 'react';

import { coursesApi, groupsApi, streamsApi } from '../../api/endpoints';
import type { Stream } from '../../api/types';
import { notifyApiError, notifySuccess } from '../../lib/notify';

export function StreamsPage() {
    const qc = useQueryClient();

    const coursesQuery = useQuery({ queryKey: ['courses'], queryFn: () => coursesApi.list() });
    const groupsQuery = useQuery({ queryKey: ['groups'], queryFn: () => groupsApi.list() });
    const [courseId, setCourseId] = useState<string | null>(null);

    const streamsQuery = useQuery({
        queryKey: ['streams', courseId],
        queryFn: () => streamsApi.listForCourse(courseId!),
        enabled: !!courseId,
    });

    const [editOpen, { open: openEdit, close: closeEdit }] = useDisclosure(false);
    const [newOpen, { open: openNew, close: closeNew }] = useDisclosure(false);

    const form = useForm<{ id: string; name: string; group_ids: string[] }>({
        initialValues: { id: '', name: '', group_ids: [] },
        validate: {
            name: (v) => (v.trim() ? null : 'Название обязательно'),
            group_ids: (v) => (v.length > 0 ? null : 'Выберите хотя бы одну группу'),
        },
    });

    const createMut = useMutation({
        mutationFn: () =>
            streamsApi.create({
                course_id: courseId!,
                name: form.values.name.trim(),
                group_ids: form.values.group_ids,
            }),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['streams', courseId] });
            notifySuccess('Поток создан');
            closeNew();
        },
        onError: (e) => notifyApiError(e),
    });

    const updateMut = useMutation({
        mutationFn: () =>
            streamsApi.update(form.values.id, {
                name: form.values.name.trim(),
                group_ids: form.values.group_ids,
            }),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['streams', courseId] });
            notifySuccess('Сохранено');
            closeEdit();
        },
        onError: (e) => notifyApiError(e),
    });

    const deleteMut = useMutation({
        mutationFn: (id: string) => streamsApi.delete(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: ['streams', courseId] }),
        onError: (e) => notifyApiError(e),
    });

    const groupData = useMemo(
        () => groupsQuery.data?.items.map((g) => ({ value: g.id, label: g.name })) ?? [],
        [groupsQuery.data],
    );

    const startEdit = (s: Stream) => {
        form.setValues({ id: s.id, name: s.name, group_ids: s.group_ids });
        openEdit();
    };

    useEffect(() => {
        if (!editOpen && !newOpen) form.reset();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [editOpen, newOpen]);

    return (
        <Stack gap="md">
            <Title order={2}>Потоки</Title>

            <Paper withBorder p="md" radius="md">
                <Group>
                    <Select
                        label="Курс"
                        placeholder="Выберите курс"
                        searchable
                        data={coursesQuery.data?.items.map((c) => ({
                            value: c.id,
                            label: `${c.code} · ${c.name}`,
                        })) ?? []}
                        value={courseId}
                        onChange={setCourseId}
                        w={360}
                    />
                    <Button onClick={openNew} disabled={!courseId} mt="lg">
                        Новый поток
                    </Button>
                </Group>
            </Paper>

            {!courseId && <Alert variant="light">Выберите курс, чтобы увидеть его потоки.</Alert>}

            {courseId && streamsQuery.isLoading && <Loader />}

            {courseId && streamsQuery.data && (
                <Paper withBorder radius="md">
                    <Table striped highlightOnHover verticalSpacing="sm">
                        <Table.Thead>
                            <Table.Tr>
                                <Table.Th>Название</Table.Th>
                                <Table.Th>Групп</Table.Th>
                                <Table.Th style={{ width: 120 }} />
                            </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody>
                            {streamsQuery.data.items.map((s) => (
                                <Table.Tr key={s.id}>
                                    <Table.Td>{s.name}</Table.Td>
                                    <Table.Td>{s.group_ids.length}</Table.Td>
                                    <Table.Td>
                                        <Group gap="xs">
                                            <Button size="xs" variant="subtle" onClick={() => startEdit(s)}>
                                                Ред.
                                            </Button>
                                            <ActionIcon
                                                color="red"
                                                variant="subtle"
                                                onClick={() => {
                                                    if (confirm(`Удалить поток «${s.name}»?`))
                                                        deleteMut.mutate(s.id);
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

            <Modal opened={newOpen} onClose={closeNew} title="Новый поток" centered>
                <form onSubmit={form.onSubmit(() => createMut.mutate())}>
                    <Stack>
                        <TextInput label="Название" {...form.getInputProps('name')} />
                        <MultiSelect
                            label="Группы"
                            searchable
                            data={groupData}
                            {...form.getInputProps('group_ids')}
                        />
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

            <Modal opened={editOpen} onClose={closeEdit} title="Редактирование потока" centered>
                <form onSubmit={form.onSubmit(() => updateMut.mutate())}>
                    <Stack>
                        <TextInput label="Название" {...form.getInputProps('name')} />
                        <MultiSelect
                            label="Группы"
                            searchable
                            data={groupData}
                            {...form.getInputProps('group_ids')}
                        />
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
