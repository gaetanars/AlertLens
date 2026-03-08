<script lang="ts">
	import { goto } from '$app/navigation';
	import { login } from '$lib/api/auth';
	import { authStore } from '$lib/stores/auth';
	import { toast } from 'svelte-sonner';
	import { Bell, Lock } from 'lucide-svelte';

	let password = $state('');
	let loading = $state(false);
	let error = $state('');

	async function submit(e: SubmitEvent) {
		e.preventDefault();
		loading = true; error = '';
		try {
			const res = await login(password);
			authStore.setToken(res.token, res.expires_at, res.role);
			const roleLabel = res.role ? ` as ${res.role}` : '';
			toast.success(`Signed in${roleLabel}`);
			goto('/alerts');
		} catch (err) {
			error = err instanceof Error ? err.message : 'Incorrect password';
		} finally {
			loading = false;
		}
	}
</script>

<div class="flex min-h-[60vh] items-center justify-center">
	<div class="w-full max-w-sm">
		<div class="text-center mb-8">
			<div class="flex justify-center mb-3">
				<div class="p-3 rounded-full bg-primary/10">
					<Bell class="h-8 w-8 text-primary" />
				</div>
			</div>
			<h1 class="text-2xl font-bold">AlertLens Admin</h1>
			<p class="text-sm text-muted-foreground mt-1">Sign in to access admin mode</p>
		</div>

		<form onsubmit={submit} class="space-y-4">
			<div>
				<label for="password" class="text-sm font-medium mb-1 block">Password</label>
				<div class="relative">
					<Lock class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
					<input
						id="password"
						type="password"
						bind:value={password}
						placeholder="••••••••"
						class="w-full pl-10 pr-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-ring"
						autofocus
					/>
				</div>
			</div>

			{#if error}
				<p class="text-sm text-destructive">{error}</p>
			{/if}

			<button
				type="submit"
				disabled={loading || !password}
				class="w-full py-2 rounded-md bg-primary text-primary-foreground font-medium hover:bg-primary/90 disabled:opacity-50 transition-colors"
			>
				{loading ? 'Signing in...' : 'Sign in'}
			</button>
		</form>
	</div>
</div>
