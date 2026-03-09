<!--
  AlertKanban.svelte — Kanban board of alert groups.

  Can work in two modes:
  1. Server-grouped: accepts groups: AlertGroup[] from the API response
     (used when group_by is sent to the backend).
  2. Client-grouped: accepts alerts: Alert[] and groupByLabel,
     and groups them client-side by severity (default) or arbitrary label.

  When groups are provided, they take precedence over alerts + groupByLabel.
-->
<script lang="ts">
	import type { Alert, AlertGroup } from '$lib/api/types';
	import AlertCard from './AlertCard.svelte';
	import { SEVERITY_ORDER, type Severity } from '$lib/utils/severity';

	let {
		alerts = [],
		groups = [],
		groupByLabel = 'severity',
		onSilence,
		onAck
	}: {
		alerts?: Alert[];
		groups?: AlertGroup[];
		groupByLabel?: string;
		onSilence?: (alert: Alert) => void;
		onAck?: (alert: Alert) => void;
	} = $props();

	// ─── Column styling ───────────────────────────────────────────────────────

	const COLUMN_HEADER: Record<Severity, string> = {
		critical: 'bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-200',
		warning:  'bg-yellow-100 text-yellow-800 dark:bg-yellow-950 dark:text-yellow-200',
		info:     'bg-blue-100 text-blue-800 dark:bg-blue-950 dark:text-blue-200',
		none:     'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200'
	};

	const STATUS_HEADER: Record<string, string> = {
		active:        'bg-green-100 text-green-800 dark:bg-green-950 dark:text-green-200',
		suppressed:    'bg-orange-100 text-orange-800 dark:bg-orange-950 dark:text-orange-200',
		unprocessed:   'bg-purple-100 text-purple-800 dark:bg-purple-950 dark:text-purple-200',
	};

	function columnHeaderClass(key: string): string {
		// severity column?
		if (key in COLUMN_HEADER) return COLUMN_HEADER[key as Severity];
		// status column?
		if (key in STATUS_HEADER) return STATUS_HEADER[key];
		return 'bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200';
	}

	// ─── Derived columns ──────────────────────────────────────────────────────

	// Priority: server-provided groups > client-side grouping.
	const columns = $derived.by((): Array<{ key: string; label: string; alerts: Alert[] }> => {
		// Use server-side groups when provided.
		if (groups.length > 0 || (alerts.length === 0 && groups !== undefined)) {
			// When groups is explicitly passed (even empty), prefer it.
			return groups.map((g) => {
				const labelVal = g.labels[groupByLabel] ?? Object.values(g.labels)[0] ?? '(none)';
				return { key: labelVal, label: labelVal, alerts: g.alerts };
			});
		}

		// Client-side grouping fallback.
		if (groupByLabel === 'severity') {
			return SEVERITY_ORDER.map((sev) => ({
				key: sev,
				label: sev === 'none' ? 'Other' : sev.charAt(0).toUpperCase() + sev.slice(1),
				alerts: alerts.filter((a) => {
					const s = (a.labels['severity']?.toLowerCase() as Severity) ?? 'none';
					const col = SEVERITY_ORDER.includes(s) ? s : 'none';
					return col === sev;
				})
			}));
		}

		// Dynamic grouping by arbitrary label.
		const map = new Map<string, Alert[]>();
		for (const alert of alerts) {
			const key = alert.labels[groupByLabel] ?? alert.alertmanager ?? '(none)';
			if (!map.has(key)) map.set(key, []);
			map.get(key)!.push(alert);
		}
		return [...map.entries()].map(([key, a]) => ({ key, label: key, alerts: a }));
	});
</script>

<div class="grid grid-cols-1 sm:grid-cols-2 xl:grid-cols-4 gap-4">
	{#each columns as col (col.key)}
		<div class="flex flex-col gap-2">
			<!-- Column header -->
			<div
				class="flex items-center justify-between px-3 py-2 rounded-lg {columnHeaderClass(col.key)}"
			>
				<span class="font-semibold text-sm capitalize">{col.label}</span>
				<span class="text-sm font-bold tabular-nums">{col.alerts.length}</span>
			</div>

			<!-- Column cards -->
			<div class="flex flex-col gap-2 min-h-[4rem]">
				{#each col.alerts as alert (alert.fingerprint)}
					<AlertCard {alert} {onSilence} {onAck} />
				{/each}
				{#if col.alerts.length === 0}
					<div
						class="flex items-center justify-center h-16 text-sm text-muted-foreground border-2 border-dashed rounded-lg"
					>
						No alerts
					</div>
				{/if}
			</div>
		</div>
	{/each}

	{#if columns.length === 0}
		<div class="col-span-full flex items-center justify-center h-32 text-sm text-muted-foreground border-2 border-dashed rounded-lg">
			No alerts
		</div>
	{/if}
</div>
