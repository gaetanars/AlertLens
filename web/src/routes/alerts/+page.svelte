<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		filteredAlerts, viewMode, alertsLoading, alertsError,
		loadAlerts, selectedFingerprints, groupByLabel
	} from '$lib/stores/alerts';
	import AlertFilters from '$lib/components/alerts/AlertFilters.svelte';
	import AlertKanban from '$lib/components/alerts/AlertKanban.svelte';
	import AlertTable from '$lib/components/alerts/AlertTable.svelte';
	import AlertBulkActions from '$lib/components/alerts/AlertBulkActions.svelte';
	import SilenceForm from '$lib/components/silences/SilenceForm.svelte';
	import AckForm from '$lib/components/silences/AckForm.svelte';
	import type { Alert, Matcher } from '$lib/api/types';

	let silenceAlert = $state<Alert | null>(null);
	let ackAlert = $state<Alert | null>(null);
	let bulkSilenceMatchers = $state<Matcher[]>([]);
	let bulkAckMatchers = $state<Matcher[]>([]);
	let showBulkSilence = $state(false);
	let showBulkAck = $state(false);
	let modal: 'silence' | 'ack' | 'bulk-silence' | 'bulk-ack' | null = $state(null);

	let interval: ReturnType<typeof setInterval>;

	onMount(() => {
		loadAlerts();
		interval = setInterval(loadAlerts, 30_000);
	});

	onDestroy(() => clearInterval(interval));

	function openSilence(alert: Alert) { silenceAlert = alert; modal = 'silence'; }
	function openAck(alert: Alert)     { ackAlert = alert;     modal = 'ack'; }
	function closeModal() { modal = null; silenceAlert = null; ackAlert = null; selectedFingerprints.set(new Set()); }

	function openBulkSilence(matchers: Matcher[]) {
		bulkSilenceMatchers = matchers; modal = 'bulk-silence';
	}
	function openBulkAck(matchers: Matcher[]) {
		bulkAckMatchers = matchers; modal = 'bulk-ack';
	}
</script>

<!-- Page header -->
<div class="flex items-center justify-between mb-4">
	<h1 class="text-xl font-bold">Alertes actives</h1>
	{#if $alertsLoading}
		<span class="text-sm text-muted-foreground animate-pulse">Chargement…</span>
	{/if}
</div>

{#if $alertsError}
	<div class="mb-4 p-3 rounded-md bg-destructive/10 text-destructive text-sm">{$alertsError}</div>
{/if}

<AlertFilters onRefresh={loadAlerts} />

<!-- Alert views -->
{#if $viewMode === 'kanban'}
	<AlertKanban alerts={$filteredAlerts} groupByLabel={$groupByLabel} onSilence={openSilence} onAck={openAck} />
{:else}
	<AlertTable alerts={$filteredAlerts} onSilence={openSilence} onAck={openAck} />
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
