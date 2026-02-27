import { authStore } from '$lib/stores/auth';
import { get } from 'svelte/store';

const BASE = '/api';

export class ApiError extends Error {
	constructor(
		public status: number,
		message: string
	) {
		super(message);
		this.name = 'ApiError';
	}
}

async function request<T>(path: string, init: RequestInit = {}): Promise<T> {
	const auth = get(authStore);
	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		Accept: 'application/json',
		...(init.headers as Record<string, string>)
	};
	if (auth.token) {
		headers['Authorization'] = `Bearer ${auth.token}`;
	}

	const res = await fetch(BASE + path, { ...init, headers });

	if (!res.ok) {
		let msg = res.statusText;
		try {
			const body = await res.json();
			msg = body.error ?? msg;
		} catch {
			// ignore parse errors
		}
		throw new ApiError(res.status, msg);
	}

	const text = await res.text();
	if (!text) return undefined as T;
	return JSON.parse(text) as T;
}

export const api = {
	get: <T>(path: string) => request<T>(path),
	post: <T>(path: string, body: unknown) =>
		request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
	put: <T>(path: string, body: unknown) =>
		request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
	delete: <T>(path: string) => request<T>(path, { method: 'DELETE' })
};
