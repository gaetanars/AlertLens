<script lang="ts">
	import type { Alert } from '$lib/api/types';
	import { getSeverity, SEVERITY_BADGE } from '$lib/utils/severity';
	import { formatRelative } from '$lib/utils/duration';
	import { isAdmin } from '$lib/stores/auth';
	import { selectedFingerprints } from '$lib/stores/alerts';
	import { User } from 'lucide-svelte';

	let { alerts, onSilence, onAck }: {
		alerts: Alert[];
		onSilence?: (alert: Alert) => void;
		onAck?: (alert: Alert) => void;
	} = $props();

	let sortKey = $state<string>('startsAt');
	let sortAsc = $state(false);

	// QUA-03: use $derived.by() for block expressions.
	const sorted = $derived.by(() =>
		[...alerts].sort((a, b) => {
			let va: string, vb: string;
			if (sortKey === 'startsAt') {
				va = a.startsAt; vb = b.startsAt;
			} else {
				va = a.labels[sortKey] ?? ''; vb = b.labels[sortKey] ?? '';
			}
			return sortAsc ? va.localeCompare(vb) : vb.localeCompare(va);
		})
	);

	function setSort(key: string) {
		if (sortKey === key) sortAsc = !sortAsc;
		else { sortKey = key; sortAsc = true; }
	}

	function toggleAll() {
		selectedFingerprints.update((s) => {
			const all = new Set(alerts.map(a => a.fingerprint));
			const allSelected = alerts.every(a => s.has(a.fingerprint));
			return allSelected ? new Set() : all;
		});
	}

	function toggleOne(fp: string) {
		selectedFingerprints.update((s) => {
			const next = new Set(s);
			if (next.has(fp)) next.delete(fp); else next.add(fp);
			return next;
		});
	}
</script>

<div class="rounded-lg border overflow-auto">
	<table class="w-full text-sm">
		<thead class="bg-muted/50">
			<tr>
				<th class="w-10 px-3 py-2">
					<input
						type="checkbox"
						checked={alerts.length > 0 && alerts.every(a => $selectedFingerprints.has(a.fingerprint))}
						onchange={toggleAll}
						class="rounded"
					/>
				</th>
				<th class="px-3 py-2 text-left cursor-pointer hover:text-foreground" onclick={() => setSort('alertname')}>
					Alert
				</th>
				<th class="px-3 py-2 text-left cursor-pointer hover:text-foreground" onclick={() => setSort('severity')}>
					Severity
				</th>
				<th class="px-3 py-2 text-left">Instance</th>
				<th class="px-3 py-2 text-left">Key labels</th>
				<th class="px-3 py-2 text-left cursor-pointer hover:text-foreground" onclick={() => setSort('startsAt')}>
					Since
				</th>
				<th class="px-3 py-2 text-left">Ack</th>
				{#if $isAdmin}
					<th class="px-3 py-2 text-left">Actions</th>
				{/if}
			</tr>
		</thead>
		<tbody>
			{#each sorted as alert (alert.fingerprint)}
				{@const severity = getSeverity(alert.labels)}
				<tr class="border-t hover:bg-muted/30 transition-colors {$selectedFingerprints.has(alert.fingerprint) ? 'bg-primary/5' : ''}">
					<td class="px-3 py-2">
						<input
							type="checkbox"
							checked={$selectedFingerprints.has(alert.fingerprint)}
							onchange={() => toggleOne(alert.fingerprint)}
							class="rounded"
						/>
					</td>
					<td class="px-3 py-2 font-medium">{alert.labels['alertname'] ?? '—'}</td>
					<td class="px-3 py-2">
						<span class="px-2 py-0.5 rounded-full text-xs {SEVERITY_BADGE[severity]}">{severity}</span>
					</td>
					<td class="px-3 py-2 text-muted-foreground">{alert.alertmanager}</td>
					<td class="px-3 py-2">
						<div class="flex flex-wrap gap-1">
							{#each Object.entries(alert.labels).filter(([k]) => !['alertname','severity'].includes(k)).slice(0, 3) as [k, v]}
								<span class="text-xs px-1 rounded bg-muted">{k}={v}</span>
							{/each}
						</div>
					</td>
					<td class="px-3 py-2 text-muted-foreground whitespace-nowrap">{formatRelative(alert.startsAt)}</td>
					<td class="px-3 py-2">
						{#if alert.ack?.active}
							<span class="flex items-center gap-1 text-xs text-purple-700 dark:text-purple-300">
								<User class="h-3 w-3" />{alert.ack.by}
							</span>
						{/if}
					</td>
					{#if $isAdmin}
						<td class="px-3 py-2">
							<div class="flex gap-1">
								{#if !alert.ack?.active}
									<button onclick={() => onAck?.(alert)} class="text-xs px-2 py-0.5 rounded bg-purple-100 text-purple-800 hover:bg-purple-200 transition-colors">Ack</button>
								{/if}
								<button onclick={() => onSilence?.(alert)} class="text-xs px-2 py-0.5 rounded bg-orange-100 text-orange-800 hover:bg-orange-200 transition-colors">Silence</button>
							</div>
						</td>
					{/if}
				</tr>
			{/each}
			{#if alerts.length === 0}
				<tr>
					<td colspan={$isAdmin ? 8 : 7} class="px-3 py-8 text-center text-muted-foreground">No active alerts</td>
				</tr>
			{/if}
		</tbody>
	</table>
</div>
