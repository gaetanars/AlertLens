<script lang="ts">
	import { fetchRouting, matchRouting } from '$lib/api/routing';
	import RoutingTree from '$lib/components/routing/RoutingTree.svelte';
	import { instances } from '$lib/stores/alerts';
	import type { RouteNode } from '$lib/api/types';
	import { AlertCircle, RefreshCw } from 'lucide-svelte';

	let selectedInstance = $state('');
	let routeData = $state<{ alertmanager: string; route: RouteNode } | null>(null);
	let loading = $state(false);
	let error = $state<string | null>(null);
	let selectedNode = $state<RouteNode | null>(null);
	let matchedRoutes = $state<RouteNode[]>([]);
	let matchLabels = $state('');

	async function load() {
		loading = true; error = null;
		try {
			routeData = await fetchRouting(selectedInstance || undefined);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load';
		} finally {
			loading = false;
		}
	}

	async function handleNodeClick(node: RouteNode) {
		selectedNode = node;
		// Build labels from node matchers for matching simulation
		const labels = Object.fromEntries(
			(node.matchers ?? []).map(m => [m.name, m.value])
		);
		try {
			const res = await matchRouting(routeData?.alertmanager ?? '', labels);
			matchedRoutes = res.matched_routes ?? [];
		} catch {
			matchedRoutes = [];
		}
	}

	// Load on mount and reload when instance changes
	$effect(() => { selectedInstance; load(); });
</script>

<div class="flex items-center justify-between mb-4">
	<h1 class="text-xl font-bold">Routing Tree</h1>
	<div class="flex gap-2">
		<select
			bind:value={selectedInstance}
			class="px-3 py-2 rounded-md border bg-background text-sm"
		>
			<option value="">Default instance</option>
			{#each $instances as inst}
				<option value={inst.name}>{inst.name}</option>
			{/each}
		</select>
		<button onclick={load} class="p-2 rounded-md border bg-background hover:bg-muted transition-colors">
			<RefreshCw class="h-4 w-4" />
		</button>
	</div>
</div>

{#if error}
	<div class="flex items-center gap-2 p-3 rounded-md bg-destructive/10 text-destructive text-sm mb-4">
		<AlertCircle class="h-4 w-4" />
		{error}
	</div>
{/if}

{#if loading}
	<div class="py-12 text-center text-muted-foreground animate-pulse">Loading…</div>
{:else if routeData}
	<div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
		<!-- Tree (takes 2/3 of width) -->
		<div class="lg:col-span-2">
			<RoutingTree route={routeData.route} onNodeClick={handleNodeClick} />
			<p class="mt-2 text-xs text-muted-foreground text-center">Click a node to see matching alerts</p>
		</div>

		<!-- Side panel -->
		<div class="space-y-4">
			{#if selectedNode}
				<div class="p-4 rounded-lg border bg-card">
					<h3 class="font-semibold mb-2">Selected node</h3>
					<div class="space-y-1 text-sm">
						<div><span class="text-muted-foreground">Receiver:</span> <strong>{selectedNode.receiver}</strong></div>
						{#if selectedNode.group_by?.length}
							<div><span class="text-muted-foreground">Group by :</span> {selectedNode.group_by.join(', ')}</div>
						{/if}
						{#if selectedNode.group_wait}
							<div><span class="text-muted-foreground">group_wait :</span> {selectedNode.group_wait}</div>
						{/if}
						{#if selectedNode.repeat_interval}
							<div><span class="text-muted-foreground">repeat_interval :</span> {selectedNode.repeat_interval}</div>
						{/if}
						{#if selectedNode.continue}
							<div class="text-primary text-xs font-medium">continue: true</div>
						{/if}
					</div>
					{#if selectedNode.matchers?.length}
						<div class="mt-2">
							<p class="text-xs text-muted-foreground mb-1">Matchers :</p>
							<div class="flex flex-wrap gap-1">
								{#each selectedNode.matchers as m}
									<code class="text-xs px-1 rounded bg-muted">{m.name}="{m.value}"</code>
								{/each}
							</div>
						</div>
					{/if}
				</div>

				{#if matchedRoutes.length > 0}
					<div class="p-4 rounded-lg border bg-card">
						<h3 class="font-semibold mb-2">Matched routes ({matchedRoutes.length})</h3>
						<div class="space-y-1">
							{#each matchedRoutes as r, i}
								<div class="flex items-center gap-2 text-sm">
									<span class="text-muted-foreground">{i + 1}.</span>
									<span class="font-medium">{r.receiver}</span>
								</div>
							{/each}
						</div>
					</div>
				{/if}
			{:else}
				<div class="p-4 rounded-lg border bg-muted/30 text-center text-sm text-muted-foreground">
					Click a node in the tree to see its details
				</div>
			{/if}
		</div>
	</div>
{/if}
