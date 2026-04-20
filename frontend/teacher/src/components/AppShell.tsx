import { AppShell as MantineAppShell, Burger, Button, Group, Text } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { useMutation } from '@tanstack/react-query';
import { Link, Outlet, useNavigate } from 'react-router-dom';

import { authApi } from '../api/endpoints';
import { useAuthStore } from '../auth/store';

export function AppShell() {
    const [opened, { toggle }] = useDisclosure();
    const navigate = useNavigate();
    const principal = useAuthStore((s) => s.principal);
    const refreshToken = useAuthStore((s) => s.refreshToken);
    const clear = useAuthStore((s) => s.clear);

    const logout = useMutation({
        mutationFn: async () => {
            if (refreshToken) {
                try {
                    await authApi.logout(refreshToken);
                } catch {
                    // даже если API упал — всё равно чистим клиент.
                }
            }
        },
        onSettled: () => {
            clear();
            navigate('/login', { replace: true });
        },
    });

    return (
        <MantineAppShell
            header={{ height: 56 }}
            navbar={{ width: 240, breakpoint: 'sm', collapsed: { mobile: !opened } }}
            padding="md"
        >
            <MantineAppShell.Header>
                <Group h="100%" px="md" justify="space-between">
                    <Group>
                        <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
                        <Text fw={600}>Teacher · Посещаемость</Text>
                    </Group>
                    <Group>
                        <Text size="sm" c="dimmed">
                            {principal?.full_name} · {principal?.role}
                        </Text>
                        <Button
                            size="xs"
                            variant="subtle"
                            onClick={() => logout.mutate()}
                            loading={logout.isPending}
                        >
                            Выйти
                        </Button>
                    </Group>
                </Group>
            </MantineAppShell.Header>

            <MantineAppShell.Navbar p="md">
                <Button
                    component={Link}
                    to="/sessions"
                    variant="subtle"
                    justify="start"
                    fullWidth
                    mb="xs"
                >
                    Сессии
                </Button>
                <Button component={Link} to="/sessions/new" variant="light" justify="start" fullWidth>
                    + Новая сессия
                </Button>
            </MantineAppShell.Navbar>

            <MantineAppShell.Main>
                <Outlet />
            </MantineAppShell.Main>
        </MantineAppShell>
    );
}
