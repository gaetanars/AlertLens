<!--
  AlertFilters.svelte — filter toolbar for the alerts page.

  Includes:
  - Free-text / matcher query
  - Severity multi-select (critical / warning / info)
  - Status multi-select (active / suppressed / unprocessed)
  - Instance select (from loaded AM instances)
  - Group-by selector (kanban mode)
  - View toggle (kanban ↔ list)
  - Refresh button
-->
<script lang="ts">
	import {
		filterQuery,
		instanceFilter,
		severityFilter,
		statusFilter,
		groupByLabel,
		viewMode,
		instances,
		loadAlerts
	} from '$lib/stores/alerts';
	import { Search, LayoutGrid, List, RefreshCw, X } from 'lucide-svelte';
	import InstanceSelector from '$lib/components/alerts/InstanceSelector.svelte';

	let { onRefresh }: { onRefresh?: () => void } = $props();

	const SEVERITIES = ['critical', 'warning', 'info'] as const;
	const STATUSES = ['active', 'suppressed', 'unprocessed'] as const;

	const GROUP_BY_OPTIONS = [
		{ value: 'severity',     label: 'Severity' },
		{ value: 'status',       label: 'Status' },
		{ value: 'alertmanager', label: 'Alertmanager' },
		{ value: 'team',         label: 'team' },
		{ value: 'environment',  label: 'environment' },
		{ value: 'cluster',      label: 'cluster' },
		{ value: 'alertname',    label: 'alertname' }
	] as const;

	// Toggle a severity filter value.
	function toggleSeverity(sev: string) {
		severityFilter.update((cur) => {
			const next = new Set(cur);
			if (next.has(sev)) next.delete(sev);
			else next.add(sev);
			return [...next];
		});
	}

	// Toggle a status filter value.
	function toggleStatus(st: string) {
		statusFilter.update((cur) => {
			const next = new Set(cur);
			if (next.has(st)) next.delete(st);
			else next.add(st);
			return [...next];
		});
	}

	function clearAllFilters() {
		filterQuery.set('');
		instanceFilter.set('');
		severityFilter.set([]);
		statusFilter.set([]);
	}

	const hasActiveFilters = $derived(
		!!$filterQuery || !!$instanceFilter || $severityFilter.length > 0 || $statusFilter.length > 0
	);

	const SEVERITY_STYLES: Record<string, string> = {
		critical: 'bg-red-100 text-red-800 border-red-300 hover:bg-red-200 dark:bg-red-900/30 dark:text-red-300 dark:border-red-800',
		warning:  'bg-yellow-100 text-yellow-800 border-yellow-300 hover:bg-yellow-200 dark:bg-yellow-900/30 dark:text-yellow-300 dark:border-yellow-800',
		info:     'bg-blue-100 text-blue-800 border-blue-300 hover:bg-blue-200 dark:bg-blue-900/30 dark:text-blue-300 dark:border-blue-800'
	};

	const SEVERITY_ACTIVE: Record<string, string> = {
		critical: 'ring-2 ring-red-500',
		warning:  'ring-2 ring-yellow-500',
		info:     'ring-2 ring-blue-500'
	};
</script>

<div class="flex flex-col gap-3 mb-4">
	<!-- Row 1: search + instance + view toggle + refresh -->
	<div class="flex flex-wrap items-center gap-2">
		<!-- Free-text / matcher filter -->
		<div class="relative flex-1 min-w-[200px]">
			<Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground pointer-events-none" />
			<input
				type="text"
				placeholder='e.g. severity="critical", env=~"prod.*"'
				bind:value={$filterQuery}
				class="w-full pl-9 pr-3 py-2 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-ring"
			/>
		</div>

		<!-- Instance filter -->
		<InstanceSelector
			bind:value={$instanceFilter}
			instances={$instances}
			onChange={() => loadAlerts()}
		/>

		<!-- Group by (kanban mode) -->
		{#if $viewMode === 'kanban'}
			<div class="flex items-center gap-2">
				<label for="groupby" class="text-sm text-muted-foreground whitespace-nowrap">Group by</label>
				<select
					id="groupby"
					bind:value={$groupByLabel}
					class="px-3 py-1.5 rounded-md border bg-background text-sm focus:outline-none focus:ring-2 focus:ring-ring"
				>
					{#each GROUP_BY_OPTIONS as opt}
						<option value={opt.value}>{opt.label}</option>
					{/each}
				</select>
			</div>
		{/if}

		<!-- View toggle -->
		<div class="flex rounded-md border overflow-hidden" role="group" aria-label="View mode">
			<button
				onclick={() => viewMode.set('kanban')}
				aria-pressed={$viewMode === 'kanban'}
				aria-label="Kanban view"
				title="Kanban view"
				class="p-2 transition-colors {$viewMode === 'kanban'
					? 'bg-primary text-primary-foreground'
					: 'bg-background hover:bg-muted'}"
			>
				<LayoutGrid class="h-4 w-4" />
			</button>
			<button
				onclick={() => viewMode.set('list')}
				aria-pressed={$viewMode === 'list'}
				aria-label="List view"
				title="List view"
				class="p-2 transition-colors {$viewMode === 'list'
					? 'bg-primary text-primary-foreground'
					: 'bg-background hover:bg-muted'}"
			>
				<List class="h-4 w-4" />
			</button>
		</div>

		<!-- Refresh -->
		<button
			onclick={onRefresh}
			aria-label="Refresh alerts"
			title="Refresh"
			class="p-2 rounded-md border bg-background hover:bg-muted transition-colors"
		>
			<RefreshCw class="h-4 w-4" />
		</button>

		<!-- Clear filters -->
		{#if hasActiveFilters}
			<button
				onclick={clearAllFilters}
				aria-label="Clear all filters"
				class="flex items-center gap-1 px-2 py-1.5 rounded-md text-sm text-muted-foreground hover:text-foreground hover:bg-muted transition-colors border"
			>
				<X class="h-3.5 w-3.5" />
				Clear
			</button>
		{/if}
	</div>

	<!-- Row 2: severity + status chip filters -->
	<div class="flex flex-wrap items-center gap-2">
		<!-- Severity chips -->
		<span class="text-xs text-muted-foreground">Severity:</span>
		{#each SEVERITIES as sev}
			<button
				onclick={() => toggleSeverity(sev)}
				aria-pressed={$severityFilter.includes(sev)}
				aria-label="Filter {sev} severity"
				class="px-2.5 py-0.5 rounded-full text-xs border transition-all {SEVERITY_STYLES[sev]} {$severityFilter.includes(sev) ? SEVERITY_ACTIVE[sev] : 'opacity-70'}"
			>
				{sev}
			</button>
		{/each}

		<span class="text-xs text-muted-foreground ml-2">Status:</span>
		{#each STATUSES as st}
			<button
				onclick={() => toggleStatus(st)}
				aria-pressed={$statusFilter.includes(st)}
				aria-label="Filter {st} status"
				class="px-2.5 py-0.5 rounded-full text-xs border transition-all bg-background hover:bg-muted {$statusFilter.includes(st) ? 'ring-2 ring-primary bg-primary/10 font-medium' : 'opacity-70'}"
			>
				{st}
			</button>
		{/each}
	</div>
</div>
