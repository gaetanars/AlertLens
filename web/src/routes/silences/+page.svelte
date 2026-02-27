<script lang="ts">
	import { onMount } from 'svelte';
	import { silences, silencesLoading, silencesError, loadSilences } from '$lib/stores/silences';
	import SilenceList from '$lib/components/silences/SilenceList.svelte';
	import SilenceForm from '$lib/components/silences/SilenceForm.svelte';
	import { isAdmin } from '$lib/stores/auth';
	import { instances, instanceFilter } from '$lib/stores/alerts';
	import type { Silence } from '$lib/api/types';
	import { Plus, RefreshCw } from 'lucide-svelte';

	let showCreate = $state(false);
	// SPEC-03: state for the silence being edited.
	let editSilence = $state<Silence | null>(null);

	onMount(() => loadSilences());

	function openEdit(s: Silence) {
		editSilence = s;
		showCreate = false;
	}

	function closeForm() {
		showCreate = false;
		editSilence = null;
	}
</script>

<div class="flex items-center justify-between mb-4">
	<h1 class="text-xl font-bold">Silences & Acks</h1>
	<div class="flex gap-2">
		<!-- Instance filter -->
		<select
			bind:value={$instanceFilter}
			class="px-3 py-2 rounded-md border bg-background text-sm"
		>
			<option value="">All instances</option>
			{#each $instances as inst}
				<option value={inst.name}>{inst.name}</option>
			{/each}
		</select>
		<button onclick={() => loadSilences($instanceFilter || undefined)} class="p-2 rounded-md border bg-background hover:bg-muted transition-colors">
			<RefreshCw class="h-4 w-4" />
		</button>
		{#if $isAdmin}
			<button
				onclick={() => { showCreate = !showCreate; editSilence = null; }}
				class="flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm transition-colors"
			>
				<Plus class="h-4 w-4" />
				New silence
			</button>
		{/if}
	</div>
</div>

{#if showCreate || editSilence}
	<div class="mb-6 p-6 rounded-xl border bg-card shadow-sm">
		<SilenceForm {editSilence} onClose={closeForm} />
	</div>
{/if}

{#if $silencesError}
	<div class="mb-4 p-3 rounded-md bg-destructive/10 text-destructive text-sm">{$silencesError}</div>
{/if}

{#if $silencesLoading}
	<div class="py-12 text-center text-muted-foreground animate-pulse">Loading…</div>
{:else}
	<SilenceList silences={$silences} onEdit={$isAdmin ? openEdit : undefined} />
{/if}
