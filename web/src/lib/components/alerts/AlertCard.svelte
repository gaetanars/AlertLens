<script lang="ts">
	import type { Alert } from '$lib/api/types';
	import { getSeverity, SEVERITY_CLASSES, SEVERITY_BADGE } from '$lib/utils/severity';
	import { formatRelative } from '$lib/utils/duration';
	import { isAdmin } from '$lib/stores/auth';
	import { selectedFingerprints } from '$lib/stores/alerts';
	import { CheckCircle2, Clock, User } from 'lucide-svelte';

	let { alert, onSilence, onAck }: {
		alert: Alert;
		onSilence?: (alert: Alert) => void;
		onAck?: (alert: Alert) => void;
	} = $props();

	const severity = $derived(getSeverity(alert.labels));
	const isSelected = $derived($selectedFingerprints.has(alert.fingerprint));

	function toggleSelect() {
		selectedFingerprints.update((s) => {
			const next = new Set(s);
			if (next.has(alert.fingerprint)) next.delete(alert.fingerprint);
			else next.add(alert.fingerprint);
			return next;
		});
	}
</script>

<div
	class="rounded-lg p-3 shadow-sm cursor-pointer transition-all hover:shadow-md
		{SEVERITY_CLASSES[severity]}
		{isSelected ? 'ring-2 ring-primary' : ''}"
	onclick={toggleSelect}
	role="checkbox"
	aria-checked={isSelected}
	tabindex="0"
	onkeydown={(e) => e.key === ' ' && toggleSelect()}
>
	<!-- Header: alertname + instance badge -->
	<div class="flex items-start justify-between gap-2 mb-2">
		<div class="flex items-center gap-2 min-w-0">
			<span class="font-semibold text-sm truncate">
				{alert.labels['alertname'] ?? 'Unknown'}
			</span>
			{#if alert.ack?.active}
				<span class="flex-shrink-0 inline-flex items-center gap-1 px-1.5 py-0.5 rounded text-xs bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200">
					<User class="h-3 w-3" />
					{alert.ack.by}
				</span>
			{/if}
		</div>
		<div class="flex items-center gap-1 flex-shrink-0">
			<span class="text-xs px-1.5 py-0.5 rounded-full {SEVERITY_BADGE[severity]}">
				{severity}
			</span>
			<span class="text-xs px-1.5 py-0.5 rounded-full bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400">
				{alert.alertmanager}
			</span>
		</div>
	</div>

	<!-- Summary annotation -->
	{#if alert.annotations['summary']}
		<p class="text-xs text-muted-foreground mb-2 line-clamp-2">
			{alert.annotations['summary']}
		</p>
	{/if}

	<!-- Key labels (excluding alertname + severity) -->
	<div class="flex flex-wrap gap-1 mb-2">
		{#each Object.entries(alert.labels).filter(([k]) => k !== 'alertname' && k !== 'severity') as [k, v]}
			<span class="text-xs px-1 py-0.5 rounded bg-muted text-muted-foreground">
				{k}=<strong>{v}</strong>
			</span>
		{/each}
	</div>

	<!-- Footer: time + actions -->
	<div class="flex items-center justify-between">
		<span class="flex items-center gap-1 text-xs text-muted-foreground">
			<Clock class="h-3 w-3" />
			{formatRelative(alert.startsAt)}
		</span>
		{#if $isAdmin}
			<div class="flex gap-1" onclick={(e) => e.stopPropagation()} role="presentation">
				{#if !alert.ack?.active}
					<button
						onclick={() => onAck?.(alert)}
						class="text-xs px-2 py-0.5 rounded bg-purple-100 text-purple-800 hover:bg-purple-200 dark:bg-purple-900 dark:text-purple-200 transition-colors"
					>
						Ack
					</button>
				{/if}
				<button
					onclick={() => onSilence?.(alert)}
					class="text-xs px-2 py-0.5 rounded bg-orange-100 text-orange-800 hover:bg-orange-200 dark:bg-orange-900 dark:text-orange-200 transition-colors"
				>
					Silence
				</button>
			</div>
		{/if}
	</div>

	<!-- Ack comment -->
	{#if alert.ack?.comment}
		<p class="mt-1 text-xs italic text-muted-foreground">"{alert.ack.comment}"</p>
	{/if}
</div>
