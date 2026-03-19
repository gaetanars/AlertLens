import { writable, derived } from 'svelte/store';
import type { UserRole } from '$lib/api/types';

interface AuthState {
	token: string | null;
	expiresAt: string | null;
	adminEnabled: boolean;
	/** Role carried by the current token ('viewer' | 'silencer' | 'config-editor' | 'admin' | '') */
	role: UserRole;
}

/** Numeric rank for each role — mirrors the backend roleRank map. */
const roleRank: Record<UserRole, number> = {
	'': 0,
	viewer: 1,
	silencer: 2,
	'config-editor': 3,
	admin: 4
};

function hasAtLeast(current: UserRole, required: UserRole): boolean {
	return roleRank[current] >= roleRank[required];
}

function createAuthStore() {
	// SEC-07: keep token in memory only — no sessionStorage to avoid XSS token theft.
	const initial: AuthState = {
		token: null,
		expiresAt: null,
		adminEnabled: false,
		role: ''
	};

	const { subscribe, set, update } = writable<AuthState>(initial);

	return {
		subscribe,
		setToken(token: string, expiresAt: string, role: UserRole) {
			update((s) => ({ ...s, token, expiresAt, role }));
		},
		setAdminEnabled(enabled: boolean) {
			update((s) => ({ ...s, adminEnabled: enabled }));
		},
		clear() {
			// Preserve adminEnabled: it reflects server configuration, not user
			// session state. Resetting it to false would hide the "Sign in" link.
			update((s) => ({ token: null, expiresAt: null, adminEnabled: s.adminEnabled, role: '' }));
		}
	};
}

export const authStore = createAuthStore();

/** Returns true only when the stored token is still valid (not expired). */
function isTokenValid($auth: AuthState): boolean {
	if (!$auth.adminEnabled) return false;
	if (!$auth.token) return false;
	if ($auth.expiresAt && new Date($auth.expiresAt) < new Date()) return false;
	return true;
}

// ─── Derived role stores ──────────────────────────────────────────────────────

/** True if the user is authenticated with ANY role. */
export const isAuthenticated = derived(authStore, ($auth) => isTokenValid($auth));

/** True if the user has at least the "silencer" role (can manage silences). */
export const canSilence = derived(authStore, ($auth) =>
	isTokenValid($auth) && hasAtLeast($auth.role, 'silencer')
);

/** True if the user has at least the "config-editor" role (can edit alertmanager config). */
export const canEditConfig = derived(authStore, ($auth) =>
	isTokenValid($auth) && hasAtLeast($auth.role, 'config-editor')
);

/** True if the user has the "admin" role. */
export const isAdmin = derived(authStore, ($auth) =>
	isTokenValid($auth) && $auth.role === 'admin'
);
