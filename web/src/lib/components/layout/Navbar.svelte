<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import { isAdmin, authStore } from '$lib/stores/auth';
	import { logout } from '$lib/api/auth';
	import { instances } from '$lib/stores/alerts';
	import { GitBranch, Volume2, Settings, LogOut, LogIn, Sun, Moon, Bell } from 'lucide-svelte';
	import { mode, toggleMode } from 'mode-watcher';

	const navItems = [
		{ href: '/alerts',   label: 'Alertes',       icon: Bell },
		{ href: '/silences', label: 'Silences',       icon: Volume2 },
		{ href: '/routing',  label: 'Routing Tree',   icon: GitBranch },
	];

	async function handleLogout() {
		await logout().catch(() => {});
		authStore.clear();
		goto('/alerts');
	}

	function toggleTheme() {
		toggleMode();
	}
</script>

<header class="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur">
	<div class="flex h-14 items-center px-4 gap-4">
		<!-- Brand -->
		<a href="/alerts" class="flex items-center gap-2 font-bold text-lg text-primary">
			<img src="/logo.png" alt="AlertLens" class="h-7 w-7" />
			AlertLens
		</a>

		<!-- Instance status dots -->
		<div class="flex gap-1 ml-2">
			{#each $instances as inst}
				<span
					title="{inst.name} {inst.healthy ? '✓' : '✗'}"
					class="h-2 w-2 rounded-full {inst.healthy ? 'bg-green-500' : 'bg-red-500'}"
				></span>
			{/each}
		</div>

		<!-- Nav -->
		<nav class="flex gap-1 ml-4">
			{#each navItems as item}
				{@const NavIcon = item.icon}
				<a
					href={item.href}
					class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-colors
						{page.url.pathname.startsWith(item.href)
							? 'bg-accent text-accent-foreground'
							: 'text-muted-foreground hover:text-foreground hover:bg-accent/50'}"
				>
					<NavIcon class="h-4 w-4" />
					{item.label}
				</a>
			{/each}
			{#if $isAdmin}
				<a
					href="/config/routing"
					class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm font-medium transition-colors
						{page.url.pathname.startsWith('/config')
							? 'bg-accent text-accent-foreground'
							: 'text-muted-foreground hover:text-foreground hover:bg-accent/50'}"
				>
					<Settings class="h-4 w-4" />
					Config
				</a>
			{/if}
		</nav>

		<!-- Spacer -->
		<div class="flex-1"></div>

		<!-- Actions -->
		<button
			onclick={toggleTheme}
			class="p-2 rounded-md text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors"
			title="Changer de thème"
		>
			{#if mode.current === 'dark'}
				<Sun class="h-4 w-4" />
			{:else}
				<Moon class="h-4 w-4" />
			{/if}
		</button>

		{#if $isAdmin}
			<button
				onclick={handleLogout}
				class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors"
			>
				<LogOut class="h-4 w-4" />
				Déconnexion
			</button>
		{:else if $authStore.adminEnabled}
			<a
				href="/login"
				class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm text-muted-foreground hover:text-foreground hover:bg-accent/50 transition-colors"
			>
				<LogIn class="h-4 w-4" />
				Admin
			</a>
		{/if}
	</div>
</header>
