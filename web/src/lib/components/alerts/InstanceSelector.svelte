<!--
  InstanceSelector.svelte — Dropdown to filter alerts by Alertmanager instance.

  Features:
  - Shows all available instances from the pool with their health status dot.
  - Emits the selected instance name via the two-way `value` binding.
  - "All instances" option (value = '') triggers a re-fetch across all instances.
  - Degraded instances (unhealthy) are shown with a warning indicator.
-->
<script lang="ts">
	import type { InstanceStatus } from '$lib/api/types';

	interface Props {
		/** Two-way binding: currently selected instance name, or '' for all. */
		value: string;
		/** List of instances from the pool (from GET /api/alertmanagers). */
		instances: InstanceStatus[];
		/** Optional: called when the selection changes. */
		onChange?: (instance: string) => void;
		/** Optional: additional CSS classes on the root element. */
		class?: string;
	}

	let { value = $bindable(''), instances = [], onChange, class: className = '' }: Props = $props();

	function handleChange(e: Event) {
		const selected = (e.target as HTMLSelectElement).value;
		value = selected;
		onChange?.(selected);
	}

	/** Returns Tailwind colour classes for the health dot. */
	function healthDotClass(inst: InstanceStatus): string {
		if (!inst.healthy) return 'bg-red-500';
		return 'bg-green-500';
	}

	/** Returns the option label, optionally appending the version. */
	function instanceLabel(inst: InstanceStatus): string {
		let label = inst.name;
		if (inst.version) label += ` (${inst.version})`;
		if (!inst.healthy) label += ' ⚠';
		return label;
	}

	const hasInstances = $derived(instances.length > 0);
	const selectedInstance = $derived(instances.find((i) => i.name === value));
</script>

<div class="relative flex items-center gap-2 {className}">
	<!-- Health indicator for the currently selected instance -->
	{#if value && selectedInstance}
		<span
			class="inline-block w-2 h-2 rounded-full flex-shrink-0 {healthDotClass(selectedInstance)}"
			aria-hidden="true"
			title={selectedInstance.healthy ? 'Healthy' : selectedInstance.error ?? 'Unhealthy'}
		></span>
	{/if}

	<select
		bind:value
		onchange={handleChange}
		aria-label="Filter by Alertmanager instance"
		disabled={!hasInstances}
		class="px-3 py-2 rounded-md border bg-background text-sm
		       focus:outline-none focus:ring-2 focus:ring-ring
		       disabled:opacity-50 disabled:cursor-not-allowed
		       min-w-[150px] max-w-[220px] truncate
		       {className}"
	>
		<option value="">All instances</option>
		{#each instances as inst (inst.name)}
			<option value={inst.name} title={inst.error ?? inst.url}>
				{instanceLabel(inst)}
			</option>
		{/each}
	</select>

	<!-- Degraded badge: shown when one or more instances are unhealthy -->
	{#if instances.some((i) => !i.healthy)}
		<span
			class="inline-flex items-center px-1.5 py-0.5 rounded text-xs font-medium bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-300 border border-yellow-300 dark:border-yellow-700 whitespace-nowrap"
			title="One or more Alertmanager instances are degraded"
			aria-label="Degraded mode: some instances are unavailable"
		>
			⚠ Degraded
		</span>
	{/if}
</div>
