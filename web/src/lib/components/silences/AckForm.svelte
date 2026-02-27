<script lang="ts">
	import type { Alert, Matcher, CreateSilenceRequest } from '$lib/api/types';
	import { instances } from '$lib/stores/alerts';
	import { createSilence } from '$lib/api/silences';
	import { loadAlerts } from '$lib/stores/alerts';
	import { toast } from 'svelte-sonner';
	import { X } from 'lucide-svelte';
	import { DURATION_PRESETS } from '$lib/utils/duration';

	let { alert = null, initialMatchers = [], onClose }: {
		alert?: Alert | null;
		initialMatchers?: Matcher[];
		onClose?: () => void;
	} = $props();

	// SVT-05: use $state so matchers are reactive when initialMatchers prop changes.
	let matchers = $state<Matcher[]>(
		alert
			? Object.entries(alert.labels).map(([name, value]) => ({ name, value, isRegex: false, isEqual: true }))
			: [...initialMatchers]
	);

	const alertmanager = $derived(alert?.alertmanager ?? $instances[0]?.name ?? '');

	let ackBy = $state('');
	let ackComment = $state('');
	let endsAt = $state(new Date(Date.now() + 8 * 3600_000).toISOString().slice(0, 16));
	let loading = $state(false);

	function applyPreset(i: number) {
		const [, e] = DURATION_PRESETS[i].getValue();
		endsAt = e.toISOString().slice(0, 16);
	}

	async function submit() {
		if (!ackBy.trim()) { toast.error('Renseignez votre nom'); return; }
		loading = true;
		try {
			const req: CreateSilenceRequest = {
				alertmanager,
				matchers,
				starts_at: new Date().toISOString(),
				ends_at: new Date(endsAt).toISOString(),
				created_by: ackBy,
				comment: ackComment,
				ack_type: 'visual',
				ack_by: ackBy,
				ack_comment: ackComment
			};
			await createSilence(req);
			toast.success(`Alerte prise en charge par ${ackBy}`);
			await loadAlerts();
			onClose?.();
		} catch (e) {
			toast.error(e instanceof Error ? e.message : 'Erreur lors de l\'ack');
		} finally {
			loading = false;
		}
	}
</script>

<div class="space-y-4">
	<div class="flex items-center justify-between">
		<h2 class="text-lg font-semibold">Prendre en charge l'alerte</h2>
		<button onclick={onClose} class="p-1 rounded hover:bg-muted"><X class="h-4 w-4" /></button>
	</div>

	{#if alert}
		<div class="p-3 rounded-md bg-muted text-sm">
			<strong>{alert.labels['alertname']}</strong> — {alert.annotations['summary'] ?? ''}
		</div>
	{/if}

	<div>
		<label class="text-sm font-medium mb-1 block">Votre nom / identifiant <span class="text-destructive">*</span></label>
		<input
			bind:value={ackBy}
			placeholder="alice"
			class="w-full px-3 py-2 rounded-md border bg-background text-sm"
			autofocus
		/>
	</div>

	<div>
		<label class="text-sm font-medium mb-1 block">Commentaire</label>
		<input
			bind:value={ackComment}
			placeholder="J'investigate..."
			class="w-full px-3 py-2 rounded-md border bg-background text-sm"
		/>
	</div>

	<div>
		<label class="text-sm font-medium mb-1 block">Durée de prise en charge</label>
		<div class="flex flex-wrap gap-2 mb-2">
			{#each DURATION_PRESETS as preset, i}
				<button onclick={() => applyPreset(i)} class="px-3 py-1 rounded-full border text-sm hover:bg-muted transition-colors">
					{preset.label}
				</button>
			{/each}
		</div>
		<input type="datetime-local" bind:value={endsAt} class="w-full px-2 py-1.5 rounded border bg-background text-sm" />
	</div>

	<button
		onclick={submit}
		disabled={loading}
		class="w-full py-2 rounded-md bg-purple-600 text-white font-medium hover:bg-purple-700 disabled:opacity-50 transition-colors"
	>
		{loading ? 'Ack en cours...' : 'Prendre en charge'}
	</button>
</div>
