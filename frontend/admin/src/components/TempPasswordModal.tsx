import { Alert, Button, Code, Group, Modal, Stack, Text } from '@mantine/core';
import { useState } from 'react';

import { copyToClipboard } from '../lib/format';
import { notifySuccess } from '../lib/notify';

// Модал показывается после create/reset-password: пароль виден один раз.
// План: "pointer-блокер на кнопку закрытия до первого копирования". Реализовано
// через disabled-состояние «Закрыть» до первого нажатия «Копировать».
export function TempPasswordModal({
    opened,
    tempPassword,
    title = 'Временный пароль',
    onClose,
}: {
    opened: boolean;
    tempPassword: string | null;
    title?: string;
    onClose: () => void;
}) {
    const [copied, setCopied] = useState(false);

    if (!tempPassword) return null;

    const handleCopy = async () => {
        const ok = await copyToClipboard(tempPassword);
        if (ok) {
            setCopied(true);
            notifySuccess('Пароль скопирован');
        }
    };

    const handleClose = () => {
        setCopied(false);
        onClose();
    };

    return (
        <Modal
            opened={opened}
            onClose={copied ? handleClose : () => {}}
            title={title}
            closeOnClickOutside={copied}
            closeOnEscape={copied}
            withCloseButton={copied}
            centered
        >
            <Stack>
                <Alert color="yellow" variant="light">
                    Пароль показывается ТОЛЬКО один раз. Скопируйте и передайте пользователю —
                    после закрытия окна восстановить его будет нельзя, можно только сбросить заново.
                </Alert>
                <Code block style={{ fontSize: 18, textAlign: 'center' }}>
                    {tempPassword}
                </Code>
                <Group justify="flex-end">
                    <Button onClick={handleCopy} variant={copied ? 'light' : 'filled'}>
                        {copied ? 'Скопировано ✓' : 'Копировать'}
                    </Button>
                    <Button onClick={handleClose} disabled={!copied} variant="default">
                        Закрыть
                    </Button>
                </Group>
                {!copied && (
                    <Text size="xs" c="dimmed">
                        Кнопка «Закрыть» активируется после копирования.
                    </Text>
                )}
            </Stack>
        </Modal>
    );
}
