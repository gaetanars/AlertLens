import { api } from './client';
import type { AuthStatus, UserRole } from './types';

export function fetchAuthStatus(): Promise<AuthStatus> {
	return api.get<AuthStatus>('/auth/status');
}

export function login(password: string): Promise<{ token: string; expires_at: string; role: UserRole }> {
	return api.post('/auth/login', { password });
}

export function logout(): Promise<void> {
	return api.post('/auth/logout', {});
}
