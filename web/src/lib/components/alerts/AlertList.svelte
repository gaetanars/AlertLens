<!--
  AlertList.svelte — interactive sortable list/table view of alerts.

  Accepts either:
  - alerts: flat Alert[] (used when caller manages grouping externally)

  Features:
  - Sortable columns (alertname, severity, startsAt)
  - Checkbox selection for bulk actions
  - Action buttons (Ack / Silence) for admin roles
  - Pagination display
-->
<script lang="ts">
	import type { Alert } from '$lib/api/types';
	import { getSeverity, SEVERITY_BADGE } from '$lib/utils/severity';
	import { formatRelative } from '$lib/utils/duration';
	import { isAdmin } from '$lib/stores/auth';
	import { selectedFingerprints } from '$lib/stores/alerts';
	import { User, ArrowUpDown, ArrowUp, ArrowDown } from 'lucide-svelte';

	let { alerts, onSilence, onAck, emptyMessage = 'No active alerts' }: {
		alerts: Alert[];
		onSilence?: (alert: Alert) => void;
		onAck?: (alert: Alert) => void;
		emptyMessage?: string;
	} = $props();

	type SortKey = 'alertname' | 'severity' | 'startsAt' | 'alertmanager';

	let sortKey = $state<SortKey>('startsAt');
	let sortAsc = $state(false);

	const SEVERITY_RANK: Record<string, number> = {
		critical: 3,
		warning: 2,
		info: 1,
		none: 0
	};

	const sorted = $derived.by(() =>
		[...alerts].sort((a, b) => {
			let cmp = 0;
			switch (sortKey) {
				case 'alertname':
					cmp = (a.labels['alertname'] ?? '').localeCompare(b.labels['alertname'] ?? '');
					break;
				case 'severity': {
					const ra = SEVERITY_RANK[a.labels['severity']?.toLowerCase() ?? 'none'] ?? 0;
					const rb = SEVERITY_RANK[b.labels['severity']?.toLowerCase() ?? 'none'] ?? 0;
					cmp = ra - rb;
					break;
				}
				case 'alertmanager':
					cmp = (a.alertmanager ?? '').localeCompare(b.alertmanager ?? '');
					break;
				case 'startsAt':
				default:
					cmp = a.startsAt.localeCompare(b.startsAt);
					break;
			}
			return sortAsc ? cmp : -cmp;
		})
	);

	function setSort(key: SortKey) {
		if (sortKey === key) {
			sortAsc = !sortAsc;
		} else {
			sortKey = key;
			sortAsc = key === 'alertname' || key === 'alertmanager';
		}
	}

	function toggleAll() {
		selectedFingerprints.update((s) => {
			const all = new Set(alerts.map((a) => a.fingerprint));
			const allSelected = alerts.every((a) => s.has(a.fingerprint));
			return allSelected ? new Set<string>() : all;
		});
	}

	function toggleOne(fp: string) {
		selectedFingerprints.update((s) => {
			const next = new Set(s);
			if (next.has(fp)) next.delete(fp);
			else next.add(fp);
			return next;
		});
	}

	function getSortIcon(key: SortKey) {
		if (sortKey !== key) return 'neutral';
		return sortAsc ? 'asc' : 'desc';
	}
</script>

<div class="rounded-lg border overflow-auto">
	<table class="w-full text-sm">
		<thead class="bg-muted/50 sticky top-0">
			<tr>
				<!-- Select-all checkbox -->
				<th class="w-10 px-3 py-2.5">
					<input
						type="checkbox"
						aria-label="Select all alerts"
						checked={alerts.length > 0 && alerts.every((a) => $selectedFingerprints.has(a.fingerprint))}
						onchange={toggleAll}
						class="rounded border-gray-300"
					/>
				</th>
				<!-- Alert name column -->
				<th
					scope="col"
					class="px-3 py-2.5 text-left cursor-pointer select-none hover:bg-muted/80 transition-colors"
					onclick={() => setSort('alertname')}
				>
					<div class="flex items-center gap-1.5 font-semibold text-xs uppercase tracking-wide text-muted-foreground">
						Alert
						{#if getSortIcon('alertname') === 'neutral'}
							<ArrowUpDown class="h-3 w-3 opacity-40" />
						{:else if getSortIcon('alertname') === 'asc'}
							<ArrowUp class="h-3 w-3" />
						{:else}
							<ArrowDown class="h-3 w-3" />
						{/if}
					</div>
				</th>
				<!-- Severity column -->
				<th
					scope="col"
					class="px-3 py-2.5 text-left cursor-pointer select-none hover:bg-muted/80 transition-colors"
					onclick={() => setSort('severity')}
				>
					<div class="flex items-center gap-1.5 font-semibold text-xs uppercase tracking-wide text-muted-foreground">
						Severity
						{#if getSortIcon('severity') === 'neutral'}
							<ArrowUpDown class="h-3 w-3 opacity-40" />
						{:else if getSortIcon('severity') === 'asc'}
							<ArrowUp class="h-3 w-3" />
						{:else}
							<ArrowDown class="h-3 w-3" />
						{/if}
					</div>
				</th>
				<!-- Status column -->
				<th scope="col" class="px-3 py-2.5 text-left font-semibold text-xs uppercase tracking-wide text-muted-foreground">
					Status
				</th>
				<!-- Instance column -->
				<th
					scope="col"
					class="px-3 py-2.5 text-left cursor-pointer select-none hover:bg-muted/80 transition-colors"
					onclick={() => setSort('alertmanager')}
				>
					<div class="flex items-center gap-1.5 font-semibold text-xs uppercase tracking-wide text-muted-foreground">
						Instance
						{#if getSortIcon('alertmanager') === 'neutral'}
							<ArrowUpDown class="h-3 w-3 opacity-40" />
						{:else if getSortIcon('alertmanager') === 'asc'}
							<ArrowUp class="h-3 w-3" />
						{:else}
							<ArrowDown class="h-3 w-3" />
						{/if}
					</div>
				</th>
				<!-- Labels column -->
				<th scope="col" class="px-3 py-2.5 text-left font-semibold text-xs uppercase tracking-wide text-muted-foreground">
					Labels
				</th>
				<!-- Since column -->
				<th
					scope="col"
					class="px-3 py-2.5 text-left cursor-pointer select-none hover:bg-muted/80 transition-colors"
					onclick={() => setSort('startsAt')}
				>
					<div class="flex items-center gap-1.5 font-semibold text-xs uppercase tracking-wide text-muted-foreground">
						Since
						{#if getSortIcon('startsAt') === 'neutral'}
							<ArrowUpDown class="h-3 w-3 opacity-40" />
						{:else if getSortIcon('startsAt') === 'asc'}
							<ArrowUp class="h-3 w-3" />
						{:else}
							<ArrowDown class="h-3 w-3" />
						{/if}
					</div>
				</th>
				<!-- Ack column -->
				<th scope="col" class="px-3 py-2.5 text-left font-semibold text-xs uppercase tracking-wide text-muted-foreground">
					Ack
				</th>
				{#if $isAdmin}
					<th scope="col" class="px-3 py-2.5 text-left font-semibold text-xs uppercase tracking-wide text-muted-foreground">
						Actions
					</th>
				{/if}
			</tr>
		</thead>
		<tbody class="divide-y divide-border">
			{#each sorted as alert (alert.fingerprint)}
				{@const severity = getSeverity(alert.labels)}
				<tr
					class="hover:bg-muted/30 transition-colors {$selectedFingerprints.has(alert.fingerprint)
						? 'bg-primary/5'
						: ''}"
				>
					<!-- Checkbox -->
					<td class="px-3 py-2.5">
						<input
							type="checkbox"
							aria-label="Select alert {alert.labels['alertname'] ?? alert.fingerprint}"
							checked={$selectedFingerprints.has(alert.fingerprint)}
							onchange={() => toggleOne(alert.fingerprint)}
							class="rounded border-gray-300"
						/>
					</td>
					<!-- Alert name -->
					<td class="px-3 py-2.5 font-medium max-w-[200px] truncate" title={alert.labels['alertname']}>
						{alert.labels['alertname'] ?? '—'}
					</td>
					<!-- Severity badge -->
					<td class="px-3 py-2.5">
						<span class="px-2 py-0.5 rounded-full text-xs font-medium {SEVERITY_BADGE[severity]}">
							{severity}
						</span>
					</td>
					<!-- Status -->
					<td class="px-3 py-2.5">
						<span class="text-xs {alert.status.state === 'active'
							? 'text-green-700 dark:text-green-400'
							: alert.status.state === 'suppressed'
							? 'text-orange-600 dark:text-orange-400'
							: 'text-muted-foreground'}">
							{alert.status.state}
						</span>
					</td>
					<!-- Instance -->
					<td class="px-3 py-2.5 text-muted-foreground text-xs">{alert.alertmanager}</td>
					<!-- Labels -->
					<td class="px-3 py-2.5">
						<div class="flex flex-wrap gap-1">
							{#each Object.entries(alert.labels)
								.filter(([k]) => !['alertname', 'severity'].includes(k))
								.slice(0, 3) as [k, v]}
								<span class="text-xs px-1.5 py-0.5 rounded bg-muted text-muted-foreground font-mono">
									{k}={v}
								</span>
							{/each}
						</div>
					</td>
					<!-- Since -->
					<td class="px-3 py-2.5 text-muted-foreground whitespace-nowrap text-xs">
						{formatRelative(alert.startsAt)}
					</td>
					<!-- Ack -->
					<td class="px-3 py-2.5">
						{#if alert.ack?.active}
							<span class="flex items-center gap-1 text-xs text-purple-700 dark:text-purple-300">
								<User class="h-3 w-3 shrink-0" />{alert.ack.by}
							</span>
						{/if}
					</td>
					<!-- Actions -->
					{#if $isAdmin}
						<td class="px-3 py-2.5">
							<div class="flex gap-1">
								{#if !alert.ack?.active}
									<button
										onclick={() => onAck?.(alert)}
										class="text-xs px-2 py-0.5 rounded bg-purple-100 text-purple-800 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-300 dark:hover:bg-purple-900/50 transition-colors"
									>
										Ack
									</button>
								{/if}
								<button
									onclick={() => onSilence?.(alert)}
									class="text-xs px-2 py-0.5 rounded bg-orange-100 text-orange-800 hover:bg-orange-200 dark:bg-orange-900/30 dark:text-orange-300 dark:hover:bg-orange-900/50 transition-colors"
								>
									Silence
								</button>
							</div>
						</td>
					{/if}
				</tr>
			{/each}
			{#if alerts.length === 0}
				<tr>
					<td
						colspan={$isAdmin ? 9 : 8}
						class="px-3 py-12 text-center text-muted-foreground text-sm"
					>
						{emptyMessage}
					</td>
				</tr>
			{/if}
		</tbody>
	</table>
</div>
