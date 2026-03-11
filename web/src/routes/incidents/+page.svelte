<script lang="ts">
	import { onMount, onDestroy } from 'svelte';
	import {
		incidents,
		filteredIncidents,
		incidentsLoading,
		incidentsError,
		incidentsTotal,
		incidentSearchQuery,
		incidentStatusFilter,
		activeIncidentCount,
		incidentsByStatus,
		loadIncidents,
		loadIncidentDetail,
		openIncident,
		selectedIncident
	} from '$lib/stores/incidents';
	import IncidentCard from '$lib/components/incidents/IncidentCard.svelte';
	import IncidentStatusBadge from '$lib/components/incidents/IncidentStatusBadge.svelte';
	import IncidentTimeline from '$lib/components/incidents/IncidentTimeline.svelte';
	import AddEventForm from '$lib/components/incidents/AddEventForm.svelte';
	import type { IncidentListItem, IncidentStatus } from '$lib/api/types';
	import { isAdmin } from '$lib/stores/auth';
	import { AlertTriangle, Plus, RefreshCw, X } from 'lucide-svelte';

	// ─── Detail panel state ───────────────────────────────────────────────────
	let detailOpen = $state(false);
	let addEventOpen = $state(false);

	// ─── Create form state ────────────────────────────────────────────────────
	let createOpen = $state(false);
	let newTitle = $state('');
	let newSeverity = $state('warning');
	let newCreatedBy = $state('');
	let newMessage = $state('');
	let creating = $state(false);
	let createError = $state<string | null>(null);

	// ─── Polling ──────────────────────────────────────────────────────────────
	let pollInterval: ReturnType<typeof setInterval>;

	onMount(() => {
		loadIncidents();
		pollInterval = setInterval(loadIncidents, 30_000);
	});

	onDestroy(() => clearInterval(pollInterval));

	// ─── Handlers ─────────────────────────────────────────────────────────────

	async function openDetail(inc: IncidentListItem) {
		await loadIncidentDetail(inc.id);
		detailOpen = true;
		addEventOpen = false;
	}

	function closeDetail() {
		detailOpen = false;
		selectedIncident.set(null);
	}

	async function handleCreate(e: SubmitEvent) {
		e.preventDefault();
		if (!newTitle.trim() || !newCreatedBy.trim()) {
			createError = 'Title and creator are required.';
			return;
		}
		creating = true;
		createError = null;
		try {
			await openIncident({
				title: newTitle.trim(),
				severity: newSeverity,
				created_by: newCreatedBy.trim(),
				initial_message: newMessage.trim() || undefined
			});
			createOpen = false;
			newTitle = '';
			newSeverity = 'warning';
			newCreatedBy = '';
			newMessage = '';
		} catch (err) {
			createError = err instanceof Error ? err.message : 'Failed to create incident.';
		} finally {
			creating = false;
		}
	}

	const STATUS_TABS: Array<{ value: IncidentStatus | ''; label: string }> = [
		{ value: '', label: 'All' },
		{ value: 'OPEN', label: 'Open' },
		{ value: 'ACK', label: 'Ack' },
		{ value: 'INVESTIGATING', label: 'Investigating' },
		{ value: 'RESOLVED', label: 'Resolved' }
	];

	const SEVERITY_OPTIONS = ['critical', 'warning', 'info'];
</script>

<svelte:head>
	<title>Incidents — AlertLens</title>
</svelte:head>

<div class="flex h-full">
	<!-- ─── Main panel ──────────────────────────────────────────────────── -->
	<main class="flex-1 overflow-auto p-4 md:p-6 space-y-4">
		<!-- Header -->
		<div class="flex items-center justify-between flex-wrap gap-3">
			<div class="flex items-center gap-3">
				<AlertTriangle class="h-5 w-5 text-destructive" />
				<h1 class="text-xl font-semibold">Incidents</h1>
				{#if $activeIncidentCount > 0}
					<span
						class="inline-flex items-center justify-center h-5 min-w-5 px-1.5
							rounded-full bg-destructive text-destructive-foreground text-xs font-bold"
					>
						{$activeIncidentCount}
					</span>
				{/if}
			</div>
			<div class="flex items-center gap-2">
				<button
					onclick={() => loadIncidents()}
					class="p-2 rounded-md hover:bg-muted transition-colors"
					title="Refresh"
					disabled={$incidentsLoading}
				>
					<RefreshCw class="h-4 w-4 {$incidentsLoading ? 'animate-spin' : ''}" />
				</button>
				{#if $isAdmin}
					<button
						onclick={() => (createOpen = true)}
						class="flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-primary
							text-primary-foreground text-sm hover:bg-primary/90 transition-colors"
					>
						<Plus class="h-4 w-4" />
						New Incident
					</button>
				{/if}
			</div>
		</div>

		<!-- Status summary row -->
		{#if !$incidentsLoading && $incidents.length > 0}
			<div class="grid grid-cols-2 sm:grid-cols-4 gap-2">
				{#each (['OPEN', 'ACK', 'INVESTIGATING', 'RESOLVED'] as const) as s}
					<button
						onclick={() => incidentStatusFilter.set($incidentStatusFilter === s ? '' : s)}
						class="rounded-lg border p-3 text-left transition-all hover:shadow-sm
							{$incidentStatusFilter === s ? 'ring-2 ring-primary' : ''}"
					>
						<p class="text-xl font-bold">{$incidentsByStatus[s].length}</p>
						<IncidentStatusBadge status={s} />
					</button>
				{/each}
			</div>
		{/if}

		<!-- Filters row -->
		<div class="flex items-center gap-2 flex-wrap">
			<!-- Status tabs -->
			<div class="flex rounded-md border overflow-hidden">
				{#each STATUS_TABS as tab}
					<button
						onclick={() => incidentStatusFilter.set(tab.value as IncidentStatus | '')}
						class="px-3 py-1.5 text-xs transition-colors
							{$incidentStatusFilter === tab.value
								? 'bg-primary text-primary-foreground'
								: 'hover:bg-muted'}"
					>
						{tab.label}
					</button>
				{/each}
			</div>

			<!-- Search -->
			<input
				type="search"
				bind:value={$incidentSearchQuery}
				placeholder="Search incidents…"
				class="rounded-md border bg-background px-3 py-1.5 text-sm flex-1 max-w-xs
					focus:outline-none focus:ring-2 focus:ring-ring placeholder:text-muted-foreground"
			/>
		</div>

		<!-- Error banner -->
		{#if $incidentsError}
			<div class="rounded-md bg-destructive/10 border border-destructive/20 px-4 py-3 text-sm text-destructive">
				{$incidentsError}
			</div>
		{/if}

		<!-- List -->
		{#if $incidentsLoading && $incidents.length === 0}
			<div class="flex justify-center py-12">
				<RefreshCw class="h-6 w-6 animate-spin text-muted-foreground" />
			</div>
		{:else if $filteredIncidents.length === 0}
			<div class="flex flex-col items-center justify-center py-16 text-muted-foreground gap-3">
				<AlertTriangle class="h-10 w-10 opacity-20" />
				<p class="text-sm">No incidents found.</p>
				{#if $isAdmin}
					<button
						onclick={() => (createOpen = true)}
						class="text-sm text-primary hover:underline"
					>
						Open the first one
					</button>
				{/if}
			</div>
		{:else}
			<div class="space-y-3">
				{#each $filteredIncidents as inc (inc.id)}
					<IncidentCard incident={inc} onClick={openDetail} />
				{/each}
			</div>
			{#if $incidentsTotal > $filteredIncidents.length}
				<p class="text-xs text-center text-muted-foreground">
					Showing {$filteredIncidents.length} of {$incidentsTotal} incidents
				</p>
			{/if}
		{/if}
	</main>

	<!-- ─── Detail side-panel ───────────────────────────────────────────── -->
	{#if detailOpen && $selectedIncident}
		{@const inc = $selectedIncident}
		<aside
			class="w-full md:w-96 border-l bg-background overflow-auto flex flex-col"
		>
			<!-- Panel header -->
			<div class="flex items-center justify-between px-4 py-3 border-b sticky top-0 bg-background z-10">
				<div class="flex items-center gap-2 min-w-0">
					<AlertTriangle class="h-4 w-4 text-destructive flex-shrink-0" />
					<h2 class="text-sm font-semibold truncate">{inc.title}</h2>
				</div>
				<div class="flex items-center gap-2 flex-shrink-0">
					<IncidentStatusBadge status={inc.status} />
					<button
						onclick={closeDetail}
						class="p-1 rounded hover:bg-muted transition-colors"
						aria-label="Close detail panel"
					>
						<X class="h-4 w-4" />
					</button>
				</div>
			</div>

			<!-- Incident metadata -->
			<div class="px-4 py-3 border-b space-y-2 text-sm">
				<div class="flex items-center gap-2 flex-wrap">
					<span class="text-muted-foreground">Severity:</span>
					<span class="font-medium capitalize">{inc.severity}</span>
				</div>
				{#if inc.alertmanager_instance}
					<div class="flex items-center gap-2">
						<span class="text-muted-foreground">Instance:</span>
						<span class="font-medium font-mono">{inc.alertmanager_instance}</span>
					</div>
				{/if}
				{#if inc.alert_fingerprint}
					<div class="flex items-center gap-2">
						<span class="text-muted-foreground">Alert:</span>
						<code class="text-xs bg-muted px-1.5 py-0.5 rounded">{inc.alert_fingerprint}</code>
					</div>
				{/if}
				{#if inc.labels && Object.keys(inc.labels).length > 0}
					<div class="flex flex-wrap gap-1 pt-1">
						{#each Object.entries(inc.labels) as [k, v]}
							<span class="text-xs px-1.5 py-0.5 rounded bg-muted text-muted-foreground">
								{k}=<strong>{v}</strong>
							</span>
						{/each}
					</div>
				{/if}
			</div>

			<!-- Timeline -->
			<div class="flex-1 overflow-auto px-4 py-4">
				<h3 class="text-xs font-semibold uppercase tracking-wider text-muted-foreground mb-4">
					Timeline
				</h3>
				<IncidentTimeline events={inc.events} />
			</div>

			<!-- Add event / action -->
			{#if $isAdmin}
				<div class="border-t px-4 py-3 bg-muted/20">
					{#if addEventOpen}
						<AddEventForm
							incidentId={inc.id}
							currentStatus={inc.status}
							onDone={async () => {
								addEventOpen = false;
								await loadIncidentDetail(inc.id);
							}}
							onCancel={() => (addEventOpen = false)}
						/>
					{:else if inc.status !== 'RESOLVED'}
						<button
							onclick={() => (addEventOpen = true)}
							class="w-full py-2 text-sm rounded-md border hover:bg-muted transition-colors"
						>
							+ Add Update
						</button>
					{/if}
				</div>
			{/if}
		</aside>
	{/if}
</div>

<!-- ─── Create incident modal ────────────────────────────────────────────── -->
{#if createOpen}
	<div
		class="fixed inset-0 z-50 flex items-center justify-center bg-background/80 backdrop-blur-sm"
		role="dialog"
		aria-modal="true"
		aria-label="Create incident"
	>
		<div class="w-full max-w-md rounded-xl border bg-card shadow-lg p-6 space-y-4">
			<div class="flex items-center justify-between">
				<h2 class="text-base font-semibold">Open New Incident</h2>
				<button onclick={() => (createOpen = false)} class="p-1 rounded hover:bg-muted">
					<X class="h-4 w-4" />
				</button>
			</div>

			<form onsubmit={handleCreate} class="space-y-4">
				<div>
					<label for="newTitle" class="block text-sm font-medium mb-1">
						Title <span class="text-destructive">*</span>
					</label>
					<input
						id="newTitle"
						type="text"
						bind:value={newTitle}
						placeholder="Brief description of the incident"
						required
						class="w-full rounded-md border bg-background px-3 py-2 text-sm
							focus:outline-none focus:ring-2 focus:ring-ring"
					/>
				</div>

				<div class="grid grid-cols-2 gap-3">
					<div>
						<label for="newSeverity" class="block text-sm font-medium mb-1">Severity</label>
						<select
							id="newSeverity"
							bind:value={newSeverity}
							class="w-full rounded-md border bg-background px-3 py-2 text-sm
								focus:outline-none focus:ring-2 focus:ring-ring"
						>
							{#each SEVERITY_OPTIONS as s}
								<option value={s}>{s}</option>
							{/each}
						</select>
					</div>
					<div>
						<label for="newCreatedBy" class="block text-sm font-medium mb-1">
							Created by <span class="text-destructive">*</span>
						</label>
						<input
							id="newCreatedBy"
							type="text"
							bind:value={newCreatedBy}
							placeholder="your name"
							required
							class="w-full rounded-md border bg-background px-3 py-2 text-sm
								focus:outline-none focus:ring-2 focus:ring-ring"
						/>
					</div>
				</div>

				<div>
					<label for="newMessage" class="block text-sm font-medium mb-1">
						Initial note
					</label>
					<textarea
						id="newMessage"
						bind:value={newMessage}
						rows={3}
						placeholder="What's happening? Any initial context…"
						class="w-full rounded-md border bg-background px-3 py-2 text-sm resize-none
							focus:outline-none focus:ring-2 focus:ring-ring"
					></textarea>
				</div>

				{#if createError}
					<p class="text-sm text-destructive">{createError}</p>
				{/if}

				<div class="flex justify-end gap-2">
					<button
						type="button"
						onclick={() => (createOpen = false)}
						class="px-4 py-2 text-sm rounded-md border hover:bg-muted transition-colors"
					>
						Cancel
					</button>
					<button
						type="submit"
						disabled={creating}
						class="px-4 py-2 text-sm rounded-md bg-destructive text-destructive-foreground
							hover:bg-destructive/90 transition-colors disabled:opacity-50"
					>
						{creating ? 'Opening…' : 'Open Incident'}
					</button>
				</div>
			</form>
		</div>
	</div>
{/if}
