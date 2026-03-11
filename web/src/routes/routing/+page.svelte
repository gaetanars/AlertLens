<script lang="ts">
	import { fetchRouting, matchRouting, fetchHubTopology } from '$lib/api/routing';
	import RoutingTree from '$lib/components/routing/RoutingTree.svelte';
	import InstanceTopology from '$lib/components/hub/InstanceTopology.svelte';
	import { instances } from '$lib/stores/alerts';
	import type { HubTopology, RouteNode, SpokeStats } from '$lib/api/types';
	import { AlertCircle, RefreshCw, GitBranch, Network } from 'lucide-svelte';

	// ─── State ────────────────────────────────────────────────────────────────

	type ViewTab = 'tree' | 'topology';

	let activeTab = $state<ViewTab>('tree');
	let selectedInstance = $state('');
	let annotateAlerts = $state(true);

	let routeData = $state<{ alertmanager: string; route: RouteNode } | null>(null);
	let hubTopology = $state<HubTopology | null>(null);

	let loading = $state(false);
	let error = $state<string | null>(null);
	let selectedNode = $state<RouteNode | null>(null);
	let matchedRoutes = $state<RouteNode[]>([]);

	// ─── Load functions ───────────────────────────────────────────────────────

	async function loadTree() {
		loading = true;
		error = null;
		try {
			routeData = await fetchRouting(selectedInstance || undefined, annotateAlerts);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load routing tree';
		} finally {
			loading = false;
		}
	}

	async function loadTopology() {
		loading = true;
		error = null;
		try {
			hubTopology = await fetchHubTopology();
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load topology';
		} finally {
			loading = false;
		}
	}

	async function load() {
		if (activeTab === 'tree') {
			await loadTree();
		} else {
			await loadTopology();
		}
	}

	async function handleNodeClick(node: RouteNode) {
		selectedNode = node;
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

	function handleSpokeClick(spoke: SpokeStats) {
		// Switch to tree view for the clicked instance.
		selectedInstance = spoke.name;
		activeTab = 'tree';
		loadTree();
	}

	// Reload when tab or instance changes.
	$effect(() => { activeTab; selectedInstance; annotateAlerts; load(); });
</script>

<!-- ─── Toolbar ─────────────────────────────────────────────────────────────── -->
<div class="flex flex-wrap items-center justify-between gap-2 mb-4">
	<h1 class="text-xl font-bold">Routing</h1>

	<div class="flex items-center gap-2">
		<!-- Tab switcher -->
		<div class="flex rounded-md border overflow-hidden text-sm">
			<button
				onclick={() => activeTab = 'tree'}
				class="px-3 py-2 flex items-center gap-1.5 transition-colors {activeTab === 'tree' ? 'bg-primary text-primary-foreground' : 'bg-background hover:bg-muted'}"
			>
				<GitBranch class="h-3.5 w-3.5" />
				Routing Tree
			</button>
			<button
				onclick={() => activeTab = 'topology'}
				class="px-3 py-2 flex items-center gap-1.5 transition-colors {activeTab === 'topology' ? 'bg-primary text-primary-foreground' : 'bg-background hover:bg-muted'}"
			>
				<Network class="h-3.5 w-3.5" />
				Hub Topology
			</button>
		</div>

		<!-- Instance picker (tree tab only) -->
		{#if activeTab === 'tree'}
			<select
				bind:value={selectedInstance}
				class="px-3 py-2 rounded-md border bg-background text-sm"
			>
				<option value="">Default instance</option>
				{#each $instances as inst}
					<option value={inst.name}>{inst.name}</option>
				{/each}
			</select>

			<!-- Annotate alerts toggle -->
			<label class="flex items-center gap-1.5 text-sm cursor-pointer select-none">
				<input
					type="checkbox"
					bind:checked={annotateAlerts}
					class="rounded"
				/>
				<span class="text-muted-foreground">Show alert counts</span>
			</label>
		{/if}

		<button
			onclick={load}
			class="p-2 rounded-md border bg-background hover:bg-muted transition-colors"
			aria-label="Refresh"
		>
			<RefreshCw class="h-4 w-4" />
		</button>
	</div>
</div>

<!-- ─── Error banner ──────────────────────────────────────────────────────────── -->
{#if error}
	<div class="flex items-center gap-2 p-3 rounded-md bg-destructive/10 text-destructive text-sm mb-4">
		<AlertCircle class="h-4 w-4" />
		{error}
	</div>
{/if}

<!-- ─── Loading state ─────────────────────────────────────────────────────────── -->
{#if loading}
	<div class="py-12 text-center text-muted-foreground animate-pulse">Loading…</div>

<!-- ─── Hub Topology tab ──────────────────────────────────────────────────────── -->
{:else if activeTab === 'topology' && hubTopology}
	<div class="space-y-4">
		<!-- Summary strip -->
		<div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
			<div class="p-3 rounded-lg border bg-card text-center">
				<p class="text-2xl font-bold">{hubTopology.hub.total_instances}</p>
				<p class="text-xs text-muted-foreground">Instances</p>
			</div>
			<div class="p-3 rounded-lg border bg-card text-center">
				<p class="text-2xl font-bold text-green-600">{hubTopology.hub.healthy_instances}</p>
				<p class="text-xs text-muted-foreground">Healthy</p>
			</div>
			<div class="p-3 rounded-lg border bg-card text-center">
				<p class="text-2xl font-bold">{hubTopology.hub.total_alerts}</p>
				<p class="text-xs text-muted-foreground">Total Alerts</p>
			</div>
			<div class="p-3 rounded-lg border bg-card text-center">
				<p class="text-2xl font-bold text-red-600">{hubTopology.hub.critical_alerts}</p>
				<p class="text-xs text-muted-foreground">Critical</p>
			</div>
		</div>

		<!-- Hub-and-spoke diagram -->
		<InstanceTopology topology={hubTopology} onSpokeClick={handleSpokeClick} />

		<!-- Spoke table -->
		<div class="rounded-lg border overflow-hidden">
			<table class="w-full text-sm">
				<thead class="bg-muted/50">
					<tr>
						<th class="px-4 py-2 text-left font-medium">Instance</th>
						<th class="px-4 py-2 text-left font-medium">Status</th>
						<th class="px-4 py-2 text-right font-medium">Total</th>
						<th class="px-4 py-2 text-right font-medium text-red-600">Critical</th>
						<th class="px-4 py-2 text-right font-medium text-amber-600">Warning</th>
						<th class="px-4 py-2 text-left font-medium">Version</th>
					</tr>
				</thead>
				<tbody>
					{#each hubTopology.spokes as spoke}
						<tr
							class="border-t hover:bg-muted/30 cursor-pointer transition-colors"
							onclick={() => handleSpokeClick(spoke)}
						>
							<td class="px-4 py-2 font-medium">{spoke.name}</td>
							<td class="px-4 py-2">
								{#if spoke.healthy}
									<span class="inline-flex items-center gap-1 text-green-600 text-xs font-medium">
										<span class="w-1.5 h-1.5 rounded-full bg-green-500 inline-block"></span>
										healthy
									</span>
								{:else}
									<span class="inline-flex items-center gap-1 text-destructive text-xs font-medium">
										<span class="w-1.5 h-1.5 rounded-full bg-destructive inline-block"></span>
										offline
									</span>
								{/if}
							</td>
							<td class="px-4 py-2 text-right">{spoke.alert_count}</td>
							<td class="px-4 py-2 text-right text-red-600">{spoke.severity_counts['critical'] ?? 0}</td>
							<td class="px-4 py-2 text-right text-amber-600">{spoke.severity_counts['warning'] ?? 0}</td>
							<td class="px-4 py-2 text-muted-foreground text-xs">{spoke.version ?? '—'}</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</div>

<!-- ─── Routing Tree tab ──────────────────────────────────────────────────────── -->
{:else if activeTab === 'tree' && routeData}
	<div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
		<!-- Tree (takes 2/3 of width) -->
		<div class="lg:col-span-2">
			<RoutingTree route={routeData.route} onNodeClick={handleNodeClick} />
			<p class="mt-2 text-xs text-muted-foreground text-center">
				Click a node to see matching routes
				{#if annotateAlerts}· Alert counts shown on each node{/if}
			</p>
		</div>

		<!-- Side panel -->
		<div class="space-y-4">
			{#if selectedNode}
				<div class="p-4 rounded-lg border bg-card">
					<h3 class="font-semibold mb-2">Selected node</h3>
					<div class="space-y-1 text-sm">
						<div><span class="text-muted-foreground">Receiver:</span> <strong>{selectedNode.receiver}</strong></div>
						{#if selectedNode.group_by?.length}
							<div><span class="text-muted-foreground">Group by:</span> {selectedNode.group_by.join(', ')}</div>
						{/if}
						{#if selectedNode.group_wait}
							<div><span class="text-muted-foreground">group_wait:</span> {selectedNode.group_wait}</div>
						{/if}
						{#if selectedNode.repeat_interval}
							<div><span class="text-muted-foreground">repeat_interval:</span> {selectedNode.repeat_interval}</div>
						{/if}
						{#if selectedNode.continue}
							<div class="text-primary text-xs font-medium">continue: true</div>
						{/if}
						{#if selectedNode.alert_count !== undefined}
							<div class="pt-1 border-t mt-1">
								<span class="text-muted-foreground">Matching alerts:</span>
								<strong class="{selectedNode.alert_count > 0 ? 'text-primary' : ''}">{selectedNode.alert_count}</strong>
							</div>
							{#if selectedNode.severity_counts && Object.keys(selectedNode.severity_counts).length > 0}
								<div class="flex flex-wrap gap-1 mt-1">
									{#each Object.entries(selectedNode.severity_counts).filter(([,v]) => v > 0) as [sev, cnt]}
										<span class="text-xs px-1.5 py-0.5 rounded-full font-medium
											{sev === 'critical' ? 'bg-red-100 text-red-700' :
											 sev === 'warning'  ? 'bg-amber-100 text-amber-700' :
											                      'bg-blue-100 text-blue-700'}">
											{sev}: {cnt}
										</span>
									{/each}
								</div>
							{/if}
						{/if}
					</div>
					{#if selectedNode.matchers?.length}
						<div class="mt-2">
							<p class="text-xs text-muted-foreground mb-1">Matchers:</p>
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
									{#if r.alert_count !== undefined}
										<span class="text-xs text-muted-foreground ml-auto">{r.alert_count} alerts</span>
									{/if}
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
