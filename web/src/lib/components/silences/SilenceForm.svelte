<script lang="ts">
	import type { Alert, Matcher, Silence, CreateSilenceRequest } from '$lib/api/types';
	import { instances } from '$lib/stores/alerts';
	import { createSilence, updateSilence } from '$lib/api/silences';
	import { loadSilences } from '$lib/stores/silences';
	import { loadAlerts } from '$lib/stores/alerts';
	import { DURATION_PRESETS } from '$lib/utils/duration';
	import { toast } from 'svelte-sonner';
	import { Plus, Trash2, X } from 'lucide-svelte';

	let { alert = null, initialMatchers = [], editSilence = null, onClose }: {
		alert?: Alert | null;
		initialMatchers?: Matcher[];
		// SPEC-03: when provided, pre-fills the form for editing an existing silence.
		editSilence?: Silence | null;
		onClose?: () => void;
	} = $props();

	const isEdit = editSilence != null;

	// Pre-fill from editSilence, alert labels, or initialMatchers.
	let matchers = $state<Matcher[]>(
		editSilence
			? editSilence.matchers.filter(m => !m.name.startsWith('alertlens_')).map(m => ({ ...m }))
			: alert
			? Object.entries(alert.labels).map(([name, value]) => ({ name, value, isRegex: false, isEqual: true }))
			: initialMatchers.length > 0
			? [...initialMatchers]
			: [{ name: '', value: '', isRegex: false, isEqual: true }]
	);

	let startsAt = $state(
		editSilence
			? new Date(editSilence.startsAt).toISOString().slice(0, 16)
			: new Date().toISOString().slice(0, 16)
	);
	let endsAt = $state(
		editSilence
			? new Date(editSilence.endsAt).toISOString().slice(0, 16)
			: new Date(Date.now() + 4 * 3600_000).toISOString().slice(0, 16)
	);
	let createdBy = $state(editSilence?.createdBy ?? '');
	let comment = $state(editSilence?.comment ?? '');
	let selectedInstance = $state(editSilence?.alertmanager ?? $instances[0]?.name ?? '');
	let loading = $state(false);

	function applyPreset(idx: number) {
		const [s, e] = DURATION_PRESETS[idx].getValue();
		startsAt = s.toISOString().slice(0, 16);
		endsAt = e.toISOString().slice(0, 16);
	}

	function addMatcher() {
		matchers = [...matchers, { name: '', value: '', isRegex: false, isEqual: true }];
	}

	function removeMatcher(i: number) {
		matchers = matchers.filter((_, idx) => idx !== i);
	}

	async function submit() {
		if (!selectedInstance) { toast.error('Select an instance'); return; }
		if (matchers.some(m => !m.name)) { toast.error('All matchers must have a name'); return; }
		loading = true;
		try {
			const req: CreateSilenceRequest = {
				alertmanager: selectedInstance,
				matchers,
				starts_at: new Date(startsAt).toISOString(),
				ends_at: new Date(endsAt).toISOString(),
				created_by: createdBy || 'alertlens',
				comment
			};
			if (isEdit && editSilence) {
				await updateSilence(editSilence.id, req);
				toast.success('Silence updated');
			} else {
				await createSilence(req);
				toast.success('Silence created');
			}
			await Promise.all([loadSilences(), loadAlerts()]);
			onClose?.();
		} catch (e) {
			toast.error(e instanceof Error ? e.message : 'Save error');
		} finally {
			loading = false;
		}
	}
</script>

<div class="space-y-4">
	<div class="flex items-center justify-between">
		<h2 class="text-lg font-semibold">{isEdit ? 'Edit silence' : 'New silence'}</h2>
		<button onclick={onClose} class="p-1 rounded hover:bg-muted"><X class="h-4 w-4" /></button>
	</div>

	<!-- Instance -->
	<div>
		<label class="text-sm font-medium mb-1 block">Instance Alertmanager</label>
		<select bind:value={selectedInstance} class="w-full px-3 py-2 rounded-md border bg-background text-sm" disabled={isEdit}>
			{#each $instances as inst}
				<option value={inst.name}>{inst.name}</option>
			{/each}
		</select>
	</div>

	<!-- Matchers -->
	<div>
		<label class="text-sm font-medium mb-1 block">Matchers</label>
		<div class="space-y-2">
			{#each matchers as matcher, i}
				<div class="flex gap-2 items-center">
					<input
						bind:value={matcher.name}
						placeholder="label"
						class="flex-1 px-2 py-1.5 rounded border bg-background text-sm"
					/>
					<select
						bind:value={matcher.isEqual}
						class="px-2 py-1.5 rounded border bg-background text-sm"
					>
						<option value={true}>=</option>
						<option value={false}>!=</option>
					</select>
					<label class="flex items-center gap-1 text-xs">
						<input type="checkbox" bind:checked={matcher.isRegex} /> regex
					</label>
					<input
						bind:value={matcher.value}
						placeholder="value"
						class="flex-1 px-2 py-1.5 rounded border bg-background text-sm"
					/>
					<button onclick={() => removeMatcher(i)} class="p-1 rounded hover:bg-muted text-muted-foreground">
						<Trash2 class="h-4 w-4" />
					</button>
				</div>
			{/each}
		</div>
		<button onclick={addMatcher} class="mt-2 flex items-center gap-1 text-sm text-primary hover:underline">
			<Plus class="h-4 w-4" /> Add matcher
		</button>
	</div>

	<!-- Duration presets -->
	<div>
		<label class="text-sm font-medium mb-1 block">Duration</label>
		<div class="flex flex-wrap gap-2 mb-2">
			{#each DURATION_PRESETS as preset, i}
				<button
					onclick={() => applyPreset(i)}
					class="px-3 py-1 rounded-full border text-sm hover:bg-muted transition-colors"
				>
					{preset.label}
				</button>
			{/each}
		</div>
		<div class="grid grid-cols-2 gap-2">
			<div>
				<label class="text-xs text-muted-foreground">Start</label>
				<input type="datetime-local" bind:value={startsAt} class="w-full px-2 py-1.5 rounded border bg-background text-sm" />
			</div>
			<div>
				<label class="text-xs text-muted-foreground">End</label>
				<input type="datetime-local" bind:value={endsAt} class="w-full px-2 py-1.5 rounded border bg-background text-sm" />
			</div>
		</div>
	</div>

	<!-- Author + comment -->
	<div class="grid grid-cols-2 gap-2">
		<div>
			<label class="text-sm font-medium mb-1 block">Created by</label>
			<input bind:value={createdBy} placeholder="your name" class="w-full px-3 py-2 rounded-md border bg-background text-sm" />
		</div>
		<div>
			<label class="text-sm font-medium mb-1 block">Comment</label>
			<input bind:value={comment} placeholder="reason..." class="w-full px-3 py-2 rounded-md border bg-background text-sm" />
		</div>
	</div>

	<button
		onclick={submit}
		disabled={loading}
		class="w-full py-2 rounded-md bg-primary text-primary-foreground font-medium hover:bg-primary/90 disabled:opacity-50 transition-colors"
	>
		{loading ? (isEdit ? 'Saving...' : 'Creating...') : (isEdit ? 'Save changes' : 'Create silence')}
	</button>
</div>
