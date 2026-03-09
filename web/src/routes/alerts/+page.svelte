<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		filteredAlerts,
		filteredGrouped,
		viewMode,
		alertsLoading,
		alertsError,
		alertsTotal,
		alertsOffset,
		alertsLimit,
		loadAlerts,
		loadInstances,
		selectedFingerprints,
		groupByLabel
	} from '$lib/stores/alerts';
	import AlertFilters from '$lib/components/alerts/AlertFilters.svelte';
	import AlertKanban from '$lib/components/alerts/AlertKanban.svelte';
	import AlertList from '$lib/components/alerts/AlertList.svelte';
	import AlertBulkActions from '$lib/components/alerts/AlertBulkActions.svelte';
	import SilenceForm from '$lib/components/silences/SilenceForm.svelte';
	import AckForm from '$lib/components/silences/AckForm.svelte';
	import type { Alert, Matcher } from '$lib/api/types';

	let silenceAlert = $state<Alert | null>(null);
	let ackAlert = $state<Alert | null>(null);
	let bulkSilenceMatchers = $state<Matcher[]>([]);
	let bulkAckMatchers = $state<Matcher[]>([]);
	let modal: 'silence' | 'ack' | 'bulk-silence' | 'bulk-ack' | null = $state(null);

	let pollingInterval: ReturnType<typeof setInterval>;

	onMount(() => {
		loadInstances();
		loadAlerts();
		// ADR-004: 30-second polling interval.
		pollingInterval = setInterval(loadAlerts, 30_000);
	});

	onDestroy(() => clearInterval(pollingInterval));

	function openSilence(alert: Alert) { silenceAlert = alert; modal = 'silence'; }
	function openAck(alert: Alert)     { ackAlert = alert;     modal = 'ack'; }
	function closeModal() {
		modal = null;
		silenceAlert = null;
		ackAlert = null;
		selectedFingerprints.set(new Set());
	}

	function openBulkSilence(matchers: Matcher[]) {
		bulkSilenceMatchers = matchers;
		modal = 'bulk-silence';
	}
	function openBulkAck(matchers: Matcher[]) {
		bulkAckMatchers = matchers;
		modal = 'bulk-ack';
	}

	// Pagination helpers.
	function prevPage() {
		alertsOffset.update((o) => Math.max(0, o - $alertsLimit));
		loadAlerts();
	}
	function nextPage() {
		alertsOffset.update((o) => o + $alertsLimit);
		loadAlerts();
	}

	const hasPrev = $derived($alertsOffset > 0);
	const hasNext = $derived($alertsOffset + $alertsLimit < $alertsTotal);
</script>

<!-- Page header -->
<div class="flex items-center justify-between mb-4">
	<div>
		<h1 class="text-xl font-bold">Active alerts</h1>
		{#if !$alertsLoading && $alertsTotal > 0}
			<p class="text-sm text-muted-foreground mt-0.5">
				{$alertsTotal} alert{$alertsTotal !== 1 ? 's' : ''} total
				{#if $alertsOffset > 0 || $alertsTotal > $alertsLimit}
					· showing {$alertsOffset + 1}–{Math.min($alertsOffset + $alertsLimit, $alertsTotal)}
				{/if}
			</p>
		{/if}
	</div>
	{#if $alertsLoading}
		<span class="text-sm text-muted-foreground animate-pulse">Refreshing…</span>
	{/if}
</div>

<!-- Error banner -->
{#if $alertsError}
	<div class="mb-4 p-3 rounded-md bg-destructive/10 text-destructive text-sm" role="alert">
		{$alertsError}
	</div>
{/if}

<!-- Filter toolbar -->
<AlertFilters onRefresh={loadAlerts} />

<!-- Alert views -->
{#if $viewMode === 'kanban'}
	<AlertKanban
		alerts={$filteredAlerts}
		groups={$filteredGrouped}
		groupByLabel={$groupByLabel}
		{onSilence}
		{onAck}
	/>
{:else}
	<AlertList
		alerts={$filteredAlerts}
		{onSilence}
		{onAck}
	/>
{/if}

<!-- Pagination bar -->
{#if $alertsTotal > $alertsLimit}
	<div class="flex items-center justify-between mt-4 text-sm">
		<button
			onclick={prevPage}
			disabled={!hasPrev}
			class="px-3 py-1.5 rounded border bg-background hover:bg-muted disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
		>
			← Previous
		</button>
		<span class="text-muted-foreground">
			Page {Math.floor($alertsOffset / $alertsLimit) + 1} of {Math.ceil($alertsTotal / $alertsLimit)}
		</span>
		<button
			onclick={nextPage}
			disabled={!hasNext}
			class="px-3 py-1.5 rounded border bg-background hover:bg-muted disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
		>
			Next →
		</button>
	</div>
{/if}

<!-- Bulk actions bar -->
<AlertBulkActions onBulkSilence={(m) => openBulkSilence(m)} onBulkAck={(m) => openBulkAck(m)} />

<!-- Modal overlay -->
{#if modal}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
		onclick={closeModal}
		role="dialog"
		aria-modal="true"
		aria-label="Alert action dialog"
	>
		<div
			class="w-full max-w-lg bg-background rounded-xl shadow-2xl p-6 mx-4"
			onclick={(e) => e.stopPropagation()}
			role="presentation"
		>
			{#if modal === 'silence'}
				<SilenceForm alert={silenceAlert} onClose={closeModal} />
			{:else if modal === 'ack'}
				<AckForm alert={ackAlert} onClose={closeModal} />
			{:else if modal === 'bulk-silence'}
				<SilenceForm initialMatchers={bulkSilenceMatchers} onClose={closeModal} />
			{:else if modal === 'bulk-ack'}
				<AckForm initialMatchers={bulkAckMatchers} onClose={closeModal} />
			{/if}
		</div>
	</div>
{/if}
