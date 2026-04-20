import { Alert, Button, Center, Paper, PasswordInput, Stack, TextInput, Title } from '@mantine/core';
import { useForm } from '@mantine/form';
import { useMutation } from '@tanstack/react-query';
import { useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';

import { ApiError } from '../api/client';
import { authApi } from '../api/endpoints';
import { useAuthStore } from '../auth/store';

export function LoginPage() {
    const navigate = useNavigate();
    const [search] = useSearchParams();
    const principal = useAuthStore((s) => s.principal);
    const setSession = useAuthStore((s) => s.setSession);

    const returnTo = search.get('return_to') || '/users';

    useEffect(() => {
        if (principal?.role === 'admin') {
            navigate(returnTo, { replace: true });
        }
    }, [principal, navigate, returnTo]);

    const form = useForm({
        initialValues: { email: '', password: '' },
        validate: {
            email: (v) => (/^\S+@\S+\.\S+$/.test(v) ? null : 'Некорректный email'),
            password: (v) => (v.length >= 1 ? null : 'Введите пароль'),
        },
    });

    const login = useMutation({
        mutationFn: async (values: { email: string; password: string }) => {
            const tokens = await authApi.login(values.email, values.password);
            useAuthStore.setState({
                accessToken: tokens.access_token,
                refreshToken: tokens.refresh_token,
            });
            const me = await authApi.me();
            return { tokens, me };
        },
        onSuccess: ({ tokens, me }) => {
            if (me.role !== 'admin') {
                useAuthStore.getState().clear();
                form.setErrors({ email: 'Этот кабинет только для администраторов.' });
                return;
            }
            setSession({
                accessToken: tokens.access_token,
                refreshToken: tokens.refresh_token,
                principal: { id: me.id, email: me.email, full_name: me.full_name, role: me.role },
            });
            navigate(returnTo, { replace: true });
        },
    });

    return (
        <Center mih="100vh" px="md">
            <Paper withBorder shadow="sm" p="xl" radius="md" w="100%" maw={420}>
                <Title order={2} mb="md">
                    Вход администратора
                </Title>

                {login.error instanceof ApiError && (
                    <Alert color="red" mb="md">
                        {login.error.code === 'invalid_credentials'
                            ? 'Неверный email или пароль.'
                            : login.error.code === 'rate_limited'
                              ? 'Слишком много попыток. Подождите минуту.'
                              : login.error.message}
                    </Alert>
                )}

                <form onSubmit={form.onSubmit((v) => login.mutate(v))}>
                    <Stack>
                        <TextInput label="Email" type="email" autoComplete="email" {...form.getInputProps('email')} />
                        <PasswordInput
                            label="Пароль"
                            autoComplete="current-password"
                            {...form.getInputProps('password')}
                        />
                        <Button type="submit" loading={login.isPending} fullWidth>
                            Войти
                        </Button>
                    </Stack>
                </form>
            </Paper>
        </Center>
    );
}
