<script lang="ts">
	import { isAuthenticated } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import { page } from '$app/state';
	import { onMount } from 'svelte';
	import { Settings, GitBranch, Clock, Radio, Save } from 'lucide-svelte';

	let { children } = $props();

	// Allow any authenticated user to reach /config/* — individual pages enforce role-based UI.
	onMount(() => {
		if (!$isAuthenticated) goto('/login');
	});

	const tabs = [
		{ href: '/config/routing',         label: 'Routing',         icon: GitBranch },
		{ href: '/config/time-intervals',  label: 'Time Intervals',  icon: Clock },
		{ href: '/config/receivers',       label: 'Receivers',       icon: Radio },
		{ href: '/config/save',            label: 'Save & Deploy',   icon: Save }
	];
</script>

<div class="space-y-4">
	<div class="flex items-center gap-2">
		<Settings class="h-5 w-5 text-primary" />
		<h1 class="text-xl font-bold">Configuration Builder</h1>
		<span class="px-2 py-0.5 rounded-full text-xs bg-primary/10 text-primary font-medium">Config Editor</span>
	</div>

	<!-- Sub-tabs -->
	<div class="flex border-b gap-1">
		{#each tabs as tab}
			{@const Icon = tab.icon}
			<a
				href={tab.href}
				class="flex items-center gap-1.5 px-4 py-2 text-sm font-medium border-b-2 transition-colors
					{page.url.pathname === tab.href
						? 'border-primary text-primary'
						: 'border-transparent text-muted-foreground hover:text-foreground hover:border-muted'}"
			>
				<Icon class="h-4 w-4" />
				{tab.label}
			</a>
		{/each}
	</div>

	{@render children()}
</div>
