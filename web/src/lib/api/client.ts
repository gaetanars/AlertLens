import { authStore } from '$lib/stores/auth';
import { get } from 'svelte/store';

const BASE = '/api';

// ─── CSRF token management ───────────────────────────────────────────────────
// The backend sets a signed csrf_token cookie (readable by JS, not HttpOnly)
// and echoes it in the X-CSRF-Token response header.
// We store the latest token in memory and include it in every mutating request.
// Bearer-authenticated requests are CSRF-exempt server-side, but we always
// send the header for defence-in-depth and to avoid 403s on the login POST
// (which has no Bearer token yet).

let csrfToken = '';

/** Update the in-memory CSRF token from a response's X-CSRF-Token header. */
function updateCSRFToken(res: Response): void {
	const headerToken = res.headers.get('X-CSRF-Token');
	if (headerToken) {
		csrfToken = headerToken;
	}
}

/** Read the CSRF token from the csrf_token cookie as a fallback. */
function readCSRFCookie(): string {
	if (typeof document === 'undefined') return '';
	const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]+)/);
	return match ? decodeURIComponent(match[1]) : '';
}

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
	const method = (init.method ?? 'GET').toUpperCase();
	const isMutating = !['GET', 'HEAD', 'OPTIONS'].includes(method);

	const headers: Record<string, string> = {
		'Content-Type': 'application/json',
		Accept: 'application/json',
		...(init.headers as Record<string, string>)
	};

	if (auth.token) {
		headers['Authorization'] = `Bearer ${auth.token}`;
	}

	// Include CSRF token for mutating requests (Bearer-exempt server-side,
	// but sending it for all routes keeps behaviour consistent and ensures
	// the unauthenticated /auth/login POST is also protected).
	if (isMutating) {
		const token = csrfToken || readCSRFCookie();
		if (token) {
			headers['X-CSRF-Token'] = token;
		}
	}

	const res = await fetch(BASE + path, { ...init, headers });

	// Always capture the latest CSRF token from the response.
	updateCSRFToken(res);

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
