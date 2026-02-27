import { writable, derived } from 'svelte/store';

interface AuthState {
	token: string | null;
	expiresAt: string | null;
	adminEnabled: boolean;
}

function createAuthStore() {
	// SEC-07: keep token in memory only — no sessionStorage to avoid XSS token theft.
	const initial: AuthState = {
		token: null,
		expiresAt: null,
		adminEnabled: false
	};

	const { subscribe, set, update } = writable<AuthState>(initial);

	return {
		subscribe,
		setToken(token: string, expiresAt: string) {
			update((s) => ({ ...s, token, expiresAt }));
		},
		setAdminEnabled(enabled: boolean) {
			update((s) => ({ ...s, adminEnabled: enabled }));
		},
		clear() {
			set({ token: null, expiresAt: null, adminEnabled: false });
		}
	};
}

export const authStore = createAuthStore();

// SVT-04: also require adminEnabled to be true before considering the session valid.
export const isAdmin = derived(authStore, ($auth) => {
	if (!$auth.adminEnabled) return false;
	if (!$auth.token) return false;
	if ($auth.expiresAt && new Date($auth.expiresAt) < new Date()) return false;
	return true;
});
