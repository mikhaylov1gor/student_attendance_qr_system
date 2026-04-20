import { Alert, Button, Center, Code, Stack } from '@mantine/core';
import { Component, type ReactNode } from 'react';

type State = { error: Error | null };

export class ErrorBoundary extends Component<{ children: ReactNode }, State> {
    state: State = { error: null };

    static getDerivedStateFromError(error: Error): State {
        return { error };
    }

    componentDidCatch(error: Error, info: unknown) {
        console.error('ErrorBoundary caught', error, info);
    }

    render() {
        if (this.state.error) {
            return (
                <Center mih="100vh" p="md">
                    <Stack maw={600}>
                        <Alert color="red" title="Что-то пошло не так">
                            Произошла непредвиденная ошибка интерфейса.
                        </Alert>
                        <Code block>{this.state.error.message}</Code>
                        <Button onClick={() => window.location.reload()}>Перезагрузить страницу</Button>
                    </Stack>
                </Center>
            );
        }
        return this.props.children;
    }
}
