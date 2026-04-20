import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type Role = 'admin' | 'teacher' | 'student';

export type Principal = {
    id: string;
    email: string;
    full_name: string;
    role: Role;
};

type AuthState = {
    accessToken: string | null;
    refreshToken: string | null;
    principal: Principal | null;
    setSession: (p: { accessToken: string; refreshToken: string; principal: Principal }) => void;
    setAccessToken: (token: string) => void;
    clear: () => void;
};

export const useAuthStore = create<AuthState>()(
    persist(
        (set) => ({
            accessToken: null,
            refreshToken: null,
            principal: null,
            setSession: ({ accessToken, refreshToken, principal }) =>
                set({ accessToken, refreshToken, principal }),
            setAccessToken: (accessToken) => set({ accessToken }),
            clear: () => set({ accessToken: null, refreshToken: null, principal: null }),
        }),
        { name: 'admin-auth' },
    ),
);
