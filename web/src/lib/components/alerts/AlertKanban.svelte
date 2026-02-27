<script lang="ts">
	import type { Alert } from '$lib/api/types';
	import AlertCard from './AlertCard.svelte';
	import { SEVERITY_ORDER, type Severity } from '$lib/utils/severity';

	let { alerts, groupByLabel = 'severity', onSilence, onAck }: {
		alerts: Alert[];
		groupByLabel?: string;
		onSilence?: (alert: Alert) => void;
		onAck?: (alert: Alert) => void;
	} = $props();

	const COLUMN_LABELS: Record<Severity, string> = {
		critical: 'Critical',
		warning:  'Warning',
		info:     'Info',
		none:     'Other'
	};

	const COLUMN_HEADER: Record<Severity, string> = {
		critical: 'bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-200',
		warning:  'bg-yellow-100 text-yellow-800 dark:bg-yellow-950 dark:text-yellow-200',
		info:     'bg-blue-100 text-blue-800 dark:bg-blue-950 dark:text-blue-200',
		none:     'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'
	};

	// QUA-03: use $derived.by() for block expressions.
	const columns = $derived.by(() => {
		if (groupByLabel === 'severity') {
			const map = new Map<Severity, Alert[]>();
			for (const sev of SEVERITY_ORDER) map.set(sev, []);
			for (const alert of alerts) {
				const s = (alert.labels['severity']?.toLowerCase() as Severity) ?? 'none';
				const col = SEVERITY_ORDER.includes(s) ? s : 'none';
				map.get(col)!.push(alert);
			}
			return { mode: 'severity' as const, map };
		}
		// QUA-06: dynamic grouping by arbitrary label.
		const map = new Map<string, Alert[]>();
		for (const alert of alerts) {
			const key = alert.labels[groupByLabel] ?? '(none)';
			if (!map.has(key)) map.set(key, []);
			map.get(key)!.push(alert);
		}
		return { mode: 'dynamic' as const, map };
	});
</script>

<div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
	{#if columns.mode === 'severity'}
		{#each SEVERITY_ORDER as severity}
			{@const colAlerts = columns.map.get(severity) ?? []}
			<div class="flex flex-col gap-2">
				<div class="flex items-center justify-between px-3 py-2 rounded-lg {COLUMN_HEADER[severity]}">
					<span class="font-semibold text-sm">{COLUMN_LABELS[severity]}</span>
					<span class="text-sm font-bold">{colAlerts.length}</span>
				</div>
				<div class="flex flex-col gap-2 min-h-[4rem]">
					{#each colAlerts as alert (alert.fingerprint)}
						<AlertCard {alert} {onSilence} {onAck} />
					{/each}
					{#if colAlerts.length === 0}
						<div class="flex items-center justify-center h-16 text-sm text-muted-foreground border-2 border-dashed rounded-lg">
							No alerts
						</div>
					{/if}
				</div>
			</div>
		{/each}
	{:else}
		{#each [...columns.map.entries()] as [key, colAlerts]}
			<div class="flex flex-col gap-2">
				<div class="flex items-center justify-between px-3 py-2 rounded-lg bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200">
					<span class="font-semibold text-sm">{key}</span>
					<span class="text-sm font-bold">{colAlerts.length}</span>
				</div>
				<div class="flex flex-col gap-2 min-h-[4rem]">
					{#each colAlerts as alert (alert.fingerprint)}
						<AlertCard {alert} {onSilence} {onAck} />
					{/each}
					{#if colAlerts.length === 0}
						<div class="flex items-center justify-center h-16 text-sm text-muted-foreground border-2 border-dashed rounded-lg">
							No alerts
						</div>
					{/if}
				</div>
			</div>
		{/each}
	{/if}
</div>
