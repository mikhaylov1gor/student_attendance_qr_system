import { Alert, Badge, Button, Group, Loader, Stack, Title } from '@mantine/core';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useNavigate, useParams } from 'react-router-dom';

import { ApiError } from '../api/client';
import { policiesApi } from '../api/endpoints';
import { PolicyForm, type PolicyFormValues } from '../components/PolicyForm';
import { notifyApiError, notifySuccess } from '../lib/notify';

export function PolicyEditPage() {
    const { id } = useParams();
    const navigate = useNavigate();
    const qc = useQueryClient();

    const policyQuery = useQuery({
        queryKey: ['policy', id],
        queryFn: () => policiesApi.get(id!),
        enabled: !!id,
    });

    const updateMut = useMutation({
        mutationFn: (values: PolicyFormValues) =>
            policiesApi.update(id!, { name: values.name.trim(), mechanisms: values.mechanisms }),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['policies'] });
            qc.invalidateQueries({ queryKey: ['policy', id] });
            notifySuccess('Сохранено');
        },
        onError: (e) => notifyApiError(e),
    });

    const setDefaultMut = useMutation({
        mutationFn: () => policiesApi.setDefault(id!),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['policies'] });
            qc.invalidateQueries({ queryKey: ['policy', id] });
            notifySuccess('Политика сделана политикой по умолчанию');
        },
        onError: (e) => notifyApiError(e),
    });

    const deleteMut = useMutation({
        mutationFn: () => policiesApi.delete(id!),
        onSuccess: () => {
            qc.invalidateQueries({ queryKey: ['policies'] });
            navigate('/policies');
        },
        onError: (e) => notifyApiError(e),
    });

    if (policyQuery.isLoading) return <Loader />;
    if (policyQuery.error instanceof ApiError) {
        return <Alert color="red">{policyQuery.error.message}</Alert>;
    }
    const p = policyQuery.data;
    if (!p) return null;

    return (
        <Stack gap="md" maw={840}>
            <Group justify="space-between">
                <Title order={2}>{p.name}</Title>
                <Group>
                    {p.is_default && <Badge color="green">default</Badge>}
                    {!p.is_default && (
                        <Button variant="light" onClick={() => setDefaultMut.mutate()} loading={setDefaultMut.isPending}>
                            Сделать default
                        </Button>
                    )}
                    <Button
                        color="red"
                        variant="subtle"
                        disabled={p.is_default}
                        onClick={() => {
                            if (confirm('Удалить политику?')) deleteMut.mutate();
                        }}
                        loading={deleteMut.isPending}
                    >
                        Удалить
                    </Button>
                </Group>
            </Group>

            <PolicyForm
                initial={{ name: p.name, mechanisms: p.mechanisms }}
                submitLabel="Сохранить"
                submitting={updateMut.isPending}
                onCancel={() => navigate('/policies')}
                onSubmit={(v) => updateMut.mutate(v)}
            />
        </Stack>
    );
}
