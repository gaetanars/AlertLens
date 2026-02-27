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
			const status = await fetchAuthStatus();
			authStore.setAdminEnabled(status.admin_enabled);
		} catch {
			// backend not reachable yet — ignore
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
