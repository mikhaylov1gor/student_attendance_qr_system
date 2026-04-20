import {
    ActionIcon,
    Button,
    Group,
    Loader,
    Modal,
    NumberInput,
    Paper,
    Stack,
    Table,
    TagsInput,
    TextInput,
    Title,
} from '@mantine/core';
import { useForm, type UseFormReturnType } from '@mantine/form';
import { useDisclosure } from '@mantine/hooks';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useEffect } from 'react';

import { classroomsApi } from '../../api/endpoints';
import type { Classroom } from '../../api/types';
import { notifyApiError, notifySuccess } from '../../lib/notify';

type FormValues = {
    id: string;
    building: string;
    room_number: string;
    latitude: number | '';
    longitude: number | '';
    radius_m: number | '';
    allowed_bssids: string[];
};

const empty: FormValues = {
    id: '',
    building: '',
    room_number: '',
    latitude: '',
    longitude: '',
    radius_m: 25,
    allowed_bssids: [],
};

export function ClassroomsPage() {
    const qc = useQueryClient();
    const classroomsQuery = useQuery({
        queryKey: ['classrooms'],
        queryFn: () => classroomsApi.list(),
    });

    const [editOpen, { open: openEdit, close: closeEdit }] = useDisclosure(false);
    const [newOpen, { open: openNew, close: closeNew }] = useDisclosure(false);

    const form = useForm<FormValues>({
        initialValues: empty,
        validate: {
            building: (v) => (v.trim() ? null : 'Здание обязательно'),
            room_number: (v) => (v.trim() ? null : 'Номер обязателен'),
            latitude: (v) => (typeof v === 'number' ? null : 'Широта обязательна'),
            longitude: (v) => (typeof v === 'number' ? null : 'Долгота обязательна'),
            radius_m: (v) => (typeof v === 'number' && v > 0 ? null : 'Радиус > 0'),
        },
    });

    const createMut = useMutation({
        mutationFn: () =>
            classroomsApi.create({
                building: form.values.building.trim(),
                room_number: form.values.room_number.trim(),
                latitude: form.values.latitude as number,
                longitude: form.values.longitude as number,
                radius_m: form.values.radius_m as number,
                allowed_bssids: form.values.allowed_bssids,
            }),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['classrooms'] });
            notifySuccess('Аудитория создана');
            closeNew();
        },
        onError: (e) => notifyApiError(e),
    });

    const updateMut = useMutation({
        mutationFn: () =>
            classroomsApi.update(form.values.id, {
                building: form.values.building.trim(),
                room_number: form.values.room_number.trim(),
                latitude: form.values.latitude as number,
                longitude: form.values.longitude as number,
                radius_m: form.values.radius_m as number,
                allowed_bssids: form.values.allowed_bssids,
            }),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['classrooms'] });
            notifySuccess('Сохранено');
            closeEdit();
        },
        onError: (e) => notifyApiError(e),
    });

    const deleteMut = useMutation({
        mutationFn: (id: string) => classroomsApi.delete(id),
        onSuccess: () => qc.invalidateQueries({ queryKey: ['classrooms'] }),
        onError: (e) => notifyApiError(e),
    });

    const startEdit = (c: Classroom) => {
        form.setValues({
            id: c.id,
            building: c.building,
            room_number: c.room_number,
            latitude: c.latitude,
            longitude: c.longitude,
            radius_m: c.radius_m,
            allowed_bssids: c.allowed_bssids,
        });
        openEdit();
    };

    useEffect(() => {
        if (!editOpen && !newOpen) form.setValues(empty);
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, [editOpen, newOpen]);

    return (
        <Stack gap="md">
            <Group justify="space-between">
                <Title order={2}>Аудитории</Title>
                <Button onClick={openNew}>Новая аудитория</Button>
            </Group>

            {classroomsQuery.isLoading && <Loader />}
            {classroomsQuery.data && (
                <Paper withBorder radius="md">
                    <Table striped highlightOnHover verticalSpacing="sm">
                        <Table.Thead>
                            <Table.Tr>
                                <Table.Th>Здание</Table.Th>
                                <Table.Th>Номер</Table.Th>
                                <Table.Th>Широта</Table.Th>
                                <Table.Th>Долгота</Table.Th>
                                <Table.Th>Радиус (м)</Table.Th>
                                <Table.Th>BSSID</Table.Th>
                                <Table.Th style={{ width: 120 }} />
                            </Table.Tr>
                        </Table.Thead>
                        <Table.Tbody>
                            {classroomsQuery.data.items.map((c) => (
                                <Table.Tr key={c.id}>
                                    <Table.Td>{c.building}</Table.Td>
                                    <Table.Td>{c.room_number}</Table.Td>
                                    <Table.Td>{c.latitude.toFixed(5)}</Table.Td>
                                    <Table.Td>{c.longitude.toFixed(5)}</Table.Td>
                                    <Table.Td>{c.radius_m}</Table.Td>
                                    <Table.Td>{c.allowed_bssids.length}</Table.Td>
                                    <Table.Td>
                                        <Group gap="xs">
                                            <Button size="xs" variant="subtle" onClick={() => startEdit(c)}>
                                                Ред.
                                            </Button>
                                            <ActionIcon
                                                color="red"
                                                variant="subtle"
                                                onClick={() => {
                                                    if (confirm(`Удалить ${c.building} ${c.room_number}?`))
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

            <ClassroomModal
                opened={newOpen}
                title="Новая аудитория"
                submitLabel="Создать"
                form={form}
                submitting={createMut.isPending}
                onSubmit={() => createMut.mutate()}
                onClose={closeNew}
            />
            <ClassroomModal
                opened={editOpen}
                title="Редактирование"
                submitLabel="Сохранить"
                form={form}
                submitting={updateMut.isPending}
                onSubmit={() => updateMut.mutate()}
                onClose={closeEdit}
            />
        </Stack>
    );
}

function ClassroomModal({
    opened,
    title,
    submitLabel,
    form,
    submitting,
    onSubmit,
    onClose,
}: {
    opened: boolean;
    title: string;
    submitLabel: string;
    form: UseFormReturnType<FormValues>;
    submitting: boolean;
    onSubmit: () => void;
    onClose: () => void;
}) {
    return (
        <Modal opened={opened} onClose={onClose} title={title} centered size="lg">
            <form onSubmit={form.onSubmit(onSubmit)}>
                <Stack>
                    <Group grow>
                        <TextInput label="Здание" {...form.getInputProps('building')} />
                        <TextInput label="Номер" {...form.getInputProps('room_number')} />
                    </Group>
                    <Group grow>
                        <NumberInput
                            label="Широта"
                            decimalScale={6}
                            {...form.getInputProps('latitude')}
                        />
                        <NumberInput
                            label="Долгота"
                            decimalScale={6}
                            {...form.getInputProps('longitude')}
                        />
                        <NumberInput
                            label="Радиус (м)"
                            min={1}
                            max={1000}
                            {...form.getInputProps('radius_m')}
                        />
                    </Group>
                    <TagsInput
                        label="Разрешённые BSSID"
                        description="Формат aa:bb:cc:dd:ee:ff, пустое значение допустимо."
                        placeholder="+ BSSID"
                        {...form.getInputProps('allowed_bssids')}
                    />
                    <Group justify="flex-end">
                        <Button variant="default" onClick={onClose}>
                            Отмена
                        </Button>
                        <Button type="submit" loading={submitting}>
                            {submitLabel}
                        </Button>
                    </Group>
                </Stack>
            </form>
        </Modal>
    );
}
