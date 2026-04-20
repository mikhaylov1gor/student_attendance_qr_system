import { AppShell as MantineAppShell, Burger, Button, Divider, Group, NavLink, Text } from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { useMutation } from '@tanstack/react-query';
import { Link, Outlet, useLocation, useNavigate } from 'react-router-dom';

import { authApi } from '../api/endpoints';
import { useAuthStore } from '../auth/store';

const LINKS = [
    { to: '/users', label: 'Пользователи' },
    { to: '/policies', label: 'Политики безопасности' },
    { to: '/audit', label: 'Журнал аудита' },
];

const CATALOG_LINKS = [
    { to: '/catalog/courses', label: 'Курсы' },
    { to: '/catalog/groups', label: 'Группы' },
    { to: '/catalog/streams', label: 'Потоки' },
    { to: '/catalog/classrooms', label: 'Аудитории' },
];

export function AppShell() {
    const [opened, { toggle }] = useDisclosure();
    const navigate = useNavigate();
    const location = useLocation();
    const principal = useAuthStore((s) => s.principal);
    const refreshToken = useAuthStore((s) => s.refreshToken);
    const clear = useAuthStore((s) => s.clear);

    const logout = useMutation({
        mutationFn: async () => {
            if (refreshToken) {
                try {
                    await authApi.logout(refreshToken);
                } catch {
                    /* ignore */
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
            navbar={{ width: 260, breakpoint: 'sm', collapsed: { mobile: !opened } }}
            padding="md"
        >
            <MantineAppShell.Header>
                <Group h="100%" px="md" justify="space-between">
                    <Group>
                        <Burger opened={opened} onClick={toggle} hiddenFrom="sm" size="sm" />
                        <Text fw={600}>Admin · Посещаемость</Text>
                    </Group>
                    <Group>
                        <Text size="sm" c="dimmed">
                            {principal?.full_name}
                        </Text>
                        <Button size="xs" variant="subtle" onClick={() => logout.mutate()} loading={logout.isPending}>
                            Выйти
                        </Button>
                    </Group>
                </Group>
            </MantineAppShell.Header>

            <MantineAppShell.Navbar p="xs">
                {LINKS.map((l) => (
                    <NavLink
                        key={l.to}
                        component={Link}
                        to={l.to}
                        label={l.label}
                        active={location.pathname.startsWith(l.to)}
                    />
                ))}
                <Divider my="xs" label="Каталог" labelPosition="left" />
                {CATALOG_LINKS.map((l) => (
                    <NavLink
                        key={l.to}
                        component={Link}
                        to={l.to}
                        label={l.label}
                        active={location.pathname.startsWith(l.to)}
                    />
                ))}
            </MantineAppShell.Navbar>

            <MantineAppShell.Main>
                <Outlet />
            </MantineAppShell.Main>
        </MantineAppShell>
    );
}
