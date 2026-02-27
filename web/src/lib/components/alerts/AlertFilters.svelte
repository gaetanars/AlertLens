<script lang="ts">
	import { filterQuery, instanceFilter, groupByLabel, viewMode, instances } from '$lib/stores/alerts';
	import { Search, LayoutGrid, List, RefreshCw } from 'lucide-svelte';

	let { onRefresh }: { onRefresh?: () => void } = $props();
</script>

<div class="flex flex-wrap items-center gap-3 mb-4">
	<!-- Filter input -->
	<div class="relative flex-1 min-w-[200px]">
		<Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
		<input
			type="text"
			placeholder='Filter: severity="critical", env=~"prod.*"'
			bind:value={$filterQuery}
			class="w-full pl-9 pr-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-ring"
		/>
	</div>

	<!-- Instance filter -->
	<select
		bind:value={$instanceFilter}
		class="px-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-ring"
	>
		<option value="">All instances</option>
		{#each $instances as inst}
			<option value={inst.name}>{inst.name}</option>
		{/each}
	</select>

	<!-- Group by (kanban mode) -->
	{#if $viewMode === 'kanban'}
		<div class="flex items-center gap-2">
			<label for="groupby" class="text-sm text-muted-foreground">Group by</label>
			<select
				id="groupby"
				bind:value={$groupByLabel}
				class="px-3 py-1.5 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-ring"
			>
				<option value="severity">severity</option>
				<option value="team">team</option>
				<option value="environment">environment</option>
				<option value="cluster">cluster</option>
				<option value="alertname">alertname</option>
			</select>
		</div>
	{/if}

	<!-- View toggle -->
	<div class="flex rounded-md border overflow-hidden">
		<button
			onclick={() => viewMode.set('kanban')}
			class="p-2 transition-colors {$viewMode === 'kanban' ? 'bg-primary text-primary-foreground' : 'bg-background hover:bg-muted'}"
			title="Kanban view"
		>
			<LayoutGrid class="h-4 w-4" />
		</button>
		<button
			onclick={() => viewMode.set('list')}
			class="p-2 transition-colors {$viewMode === 'list' ? 'bg-primary text-primary-foreground' : 'bg-background hover:bg-muted'}"
			title="List view"
		>
			<List class="h-4 w-4" />
		</button>
	</div>

	<!-- Refresh -->
	<button
		onclick={onRefresh}
		class="p-2 rounded-md border bg-background hover:bg-muted transition-colors"
		title="Refresh"
	>
		<RefreshCw class="h-4 w-4" />
	</button>
</div>
