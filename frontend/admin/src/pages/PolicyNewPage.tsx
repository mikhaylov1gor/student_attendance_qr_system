import { Stack, Title } from '@mantine/core';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';

import { policiesApi } from '../api/endpoints';
import { DEFAULT_MECHANISMS, PolicyForm } from '../components/PolicyForm';
import { notifyApiError, notifySuccess } from '../lib/notify';

export function PolicyNewPage() {
    const navigate = useNavigate();
    const qc = useQueryClient();

    const createMut = useMutation({
        mutationFn: (values: { name: string; mechanisms: typeof DEFAULT_MECHANISMS }) =>
            policiesApi.create({ name: values.name.trim(), mechanisms: values.mechanisms }),
        onSuccess: (p) => {
            qc.invalidateQueries({ queryKey: ['policies'] });
            notifySuccess('Политика создана');
            navigate(`/policies/${p.id}`);
        },
        onError: (e) => notifyApiError(e),
    });

    return (
        <Stack gap="md" maw={840}>
            <Title order={2}>Новая политика</Title>
            <PolicyForm
                initial={{ name: '', mechanisms: DEFAULT_MECHANISMS }}
                submitLabel="Создать"
                submitting={createMut.isPending}
                onCancel={() => navigate('/policies')}
                onSubmit={(v) => createMut.mutate(v)}
            />
        </Stack>
    );
}
