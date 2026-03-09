<script lang="ts">
	import '../app.css';
	import { ModeWatcher } from 'mode-watcher';
	import { Toaster } from 'svelte-sonner';
	import Navbar from '$lib/components/layout/Navbar.svelte';
	import { onMount } from 'svelte';
	import { fetchAuthStatus } from '$lib/api/auth';
	import { authStore } from '$lib/stores/auth';
	import { loadInstances } from '$lib/stores/alerts';

	let { children } = $props();

	onMount(async () => {
		try {
			// SEC-CSRF: fetchAuthStatus() is a GET request. The backend sets a
			// fresh signed csrf_token cookie AND echoes it in the X-CSRF-Token
			// response header on every safe-method response (see CSRFMiddleware).
			// The api client captures that header automatically (updateCSRFToken),
			// so after this call the in-memory CSRF token is primed and every
			// subsequent mutating request (POST/PUT/DELETE) will carry it.
			const status = await fetchAuthStatus();
			authStore.setAdminEnabled(status.admin_enabled);
			if (status.authenticated && status.role) {
				authStore.setAdminEnabled(status.admin_enabled);
			}
		} catch {
			// backend not reachable yet — ignore; the cookie fallback in
			// readCSRFCookie() will be used if the in-memory token is empty.
		}
		loadInstances();
	});
</script>

<svelte:head>
	<title>AlertLens</title>
</svelte:head>

<ModeWatcher defaultMode="light" />
<Toaster richColors position="top-right" />

<div class="min-h-screen flex flex-col">
	<Navbar />
	<main class="flex-1 container mx-auto px-4 py-6 max-w-screen-2xl">
		{@render children()}
	</main>
</div>
