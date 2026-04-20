import {
    Alert,
    Button,
    Checkbox,
    Divider,
    Group,
    NumberInput,
    Paper,
    Stack,
    TagsInput,
    TextInput,
    Title,
} from '@mantine/core';
import { useForm } from '@mantine/form';

import type { MechanismsConfig } from '../api/types';

export type PolicyFormValues = {
    name: string;
    mechanisms: MechanismsConfig;
};

export const DEFAULT_MECHANISMS: MechanismsConfig = {
    qr_ttl: { enabled: true, ttl_seconds: 10 },
    geo: { enabled: false },
    wifi: { enabled: false, required_bssids_from_classroom: true, extra_bssids: [] },
    bluetooth_beacon: { enabled: false },
};

type Props = {
    initial: PolicyFormValues;
    submitting: boolean;
    onSubmit: (values: PolicyFormValues) => void;
    onCancel: () => void;
    submitLabel: string;
};

// Форма политики — единая для create и update. Live-preview TTL справа от поля.
export function PolicyForm({ initial, submitting, onSubmit, onCancel, submitLabel }: Props) {
    const form = useForm<PolicyFormValues>({
        initialValues: initial,
        validate: {
            name: (v) => (v.trim() ? null : 'Название обязательно'),
            mechanisms: {
                qr_ttl: {
                    ttl_seconds: (v, values) => {
                        if (!values.mechanisms.qr_ttl.enabled) return null;
                        if (v < 3 || v > 120) return 'TTL должен быть в диапазоне [3, 120]';
                        return null;
                    },
                },
                geo: {
                    radius_override_m: (v, values) => {
                        if (!values.mechanisms.geo.enabled || v == null) return null;
                        if (v < 1 || v > 1000) return 'radius_override_m: [1, 1000]';
                        return null;
                    },
                },
            },
        },
    });

    return (
        <form onSubmit={form.onSubmit(onSubmit)}>
            <Stack>
                <Paper withBorder p="md" radius="md">
                    <TextInput label="Название политики" {...form.getInputProps('name')} />
                </Paper>

                <Paper withBorder p="md" radius="md">
                    <Title order={4} mb="sm">
                        Механизмы защиты
                    </Title>

                    <Stack gap="md">
                        <MechanismSection title="QR TTL — ротация по счётчику" enabledProps={form.getInputProps('mechanisms.qr_ttl.enabled', { type: 'checkbox' })}>
                            <Group grow>
                                <NumberInput
                                    label="TTL (секунд)"
                                    min={3}
                                    max={120}
                                    description="Отметка с устаревшим counter'ом → needs_review"
                                    {...form.getInputProps('mechanisms.qr_ttl.ttl_seconds')}
                                />
                                <Alert variant="light" color="blue" mt="xl">
                                    Ротация каждые <b>{form.values.mechanisms.qr_ttl.ttl_seconds}s</b>.
                                    За минуту — ≈{' '}
                                    {Math.round(60 / (form.values.mechanisms.qr_ttl.ttl_seconds || 1))}{' '}
                                    новых QR.
                                </Alert>
                            </Group>
                        </MechanismSection>

                        <Divider />

                        <MechanismSection title="Геолокация" enabledProps={form.getInputProps('mechanisms.geo.enabled', { type: 'checkbox' })}>
                            <NumberInput
                                label="radius_override_m (опционально)"
                                description="Переопределяет radius_m, заданный в аудитории."
                                placeholder="не задан"
                                min={1}
                                max={1000}
                                {...form.getInputProps('mechanisms.geo.radius_override_m')}
                            />
                        </MechanismSection>

                        <Divider />

                        <MechanismSection title="Wi-Fi (BSSID)" enabledProps={form.getInputProps('mechanisms.wifi.enabled', { type: 'checkbox' })}>
                            <Checkbox
                                label="Использовать список BSSID из аудитории"
                                {...form.getInputProps('mechanisms.wifi.required_bssids_from_classroom', {
                                    type: 'checkbox',
                                })}
                            />
                            <TagsInput
                                label="Дополнительные BSSID"
                                description="Формат aa:bb:cc:dd:ee:ff"
                                placeholder="+ BSSID"
                                {...form.getInputProps('mechanisms.wifi.extra_bssids')}
                            />
                        </MechanismSection>

                        <Divider />

                        <MechanismSection
                            title="Bluetooth-маяки (заглушка)"
                            enabledProps={form.getInputProps('mechanisms.bluetooth_beacon.enabled', {
                                type: 'checkbox',
                            })}
                        >
                            <Alert variant="light" color="gray">
                                Реализация механизма — задел на будущее. Включение сейчас не влияет
                                на поведение (no-op проверка).
                            </Alert>
                        </MechanismSection>
                    </Stack>
                </Paper>

                <Group justify="flex-end">
                    <Button variant="default" onClick={onCancel}>
                        Отмена
                    </Button>
                    <Button type="submit" loading={submitting}>
                        {submitLabel}
                    </Button>
                </Group>
            </Stack>
        </form>
    );
}

function MechanismSection({
    title,
    enabledProps,
    children,
}: {
    title: string;
    enabledProps: ReturnType<ReturnType<typeof useForm>['getInputProps']>;
    children: React.ReactNode;
}) {
    return (
        <Stack gap="sm">
            <Checkbox label={<b>{title}</b>} {...enabledProps} />
            <div style={{ paddingLeft: 28 }}>{children}</div>
        </Stack>
    );
}
