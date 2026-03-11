<script lang="ts">
	import { selectedFingerprints, alerts } from '$lib/stores/alerts';
	import { loadAlerts } from '$lib/stores/alerts';
	import { canSilence } from '$lib/stores/auth';
	import { bulkSilence, bulkAck } from '$lib/api/bulk';
	import { toast } from 'svelte-sonner';
	import { X, Volume2, User, Zap } from 'lucide-svelte';
	import type { Matcher } from '$lib/api/types';

	// QUA-04: callers use the store directly to get selectedAlerts; no need to pass them.
	let { onBulkSilence, onBulkAck }: {
		onBulkSilence?: (matchers: Matcher[]) => void;
		onBulkAck?: (matchers: Matcher[]) => void;
	} = $props();

	const count = $derived($selectedFingerprints.size);
	const selectedAlerts = $derived(
		$alerts.filter((a) => $selectedFingerprints.has(a.fingerprint))
	);

	let quickLoading = $state(false);

	// Compute the intersection of label matchers common to all selected alerts.
	// Used for the form-based "customize" path.
	function computeCommonMatchers(): Matcher[] {
		if (selectedAlerts.length === 0) return [];
		const first = selectedAlerts[0];
		const common: Matcher[] = [];
		for (const [key, val] of Object.entries(first.labels)) {
			if (selectedAlerts.every((a) => a.labels[key] === val)) {
				common.push({ name: key, value: val, isRegex: false, isEqual: true });
			}
		}
		return common;
	}

	function clearSelection() {
		selectedFingerprints.set(new Set());
	}

	/**
	 * ADR-007: Quick Silence — calls POST /api/v1/bulk with smart-merge logic.
	 * Silences all selected alerts for 1 hour without opening a form.
	 */
	async function quickSilence() {
		if (selectedAlerts.length === 0) return;
		quickLoading = true;
		try {
			const res = await bulkSilence(selectedAlerts, {
				endsAt: new Date(Date.now() + 3_600_000), // +1h
				comment: 'Bulk silenced via AlertLens'
			});
			const label = res.strategy === 'merged'
				? `Merged into ${res.count} silence${res.count !== 1 ? 's' : ''}`
				: `Created ${res.count} individual silence${res.count !== 1 ? 's' : ''}`;
			toast.success(`${count} alert${count !== 1 ? 's' : ''} silenced (1 h) · ${label}`);
			clearSelection();
			await loadAlerts();
		} catch (e) {
			toast.error(e instanceof Error ? e.message : 'Bulk silence failed');
		} finally {
			quickLoading = false;
		}
	}

	/**
	 * ADR-007: Quick Ack — calls POST /api/v1/bulk with action="ack".
	 * Visually acks all selected alerts for 1 hour without opening a form.
	 */
	async function quickAck() {
		if (selectedAlerts.length === 0) return;
		quickLoading = true;
		try {
			const res = await bulkAck(selectedAlerts, {
				endsAt: new Date(Date.now() + 3_600_000), // +1h
				comment: 'Bulk acked via AlertLens'
			});
			const label = res.strategy === 'merged'
				? `Merged into ${res.count} ack${res.count !== 1 ? 's' : ''}`
				: `Created ${res.count} individual ack${res.count !== 1 ? 's' : ''}`;
			toast.success(`${count} alert${count !== 1 ? 's' : ''} acked (1 h) · ${label}`);
			clearSelection();
			await loadAlerts();
		} catch (e) {
			toast.error(e instanceof Error ? e.message : 'Bulk ack failed');
		} finally {
			quickLoading = false;
		}
	}
</script>

{#if count > 0}
	<div class="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 flex items-center gap-3 px-4 py-3 rounded-xl shadow-lg border bg-background">
		<span class="text-sm font-medium">{count} alert{count > 1 ? 's' : ''} selected</span>
		<div class="h-4 w-px bg-border"></div>

		{#if $canSilence}
			<!-- Quick actions (ADR-007): call /api/v1/bulk directly, no form -->
			<button
				onclick={quickAck}
				disabled={quickLoading}
				class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm bg-purple-100 text-purple-800 hover:bg-purple-200 disabled:opacity-50 transition-colors"
				title="Smart-merge ack all selected alerts (1 h)"
			>
				{#if quickLoading}
					<span class="h-4 w-4 animate-spin rounded-full border-2 border-purple-800 border-t-transparent"></span>
				{:else}
					<Zap class="h-4 w-4" />
				{/if}
				Quick ack (1 h)
			</button>

			<button
				onclick={quickSilence}
				disabled={quickLoading}
				class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm bg-orange-100 text-orange-800 hover:bg-orange-200 disabled:opacity-50 transition-colors"
				title="Smart-merge silence all selected alerts (1 h)"
			>
				{#if quickLoading}
					<span class="h-4 w-4 animate-spin rounded-full border-2 border-orange-800 border-t-transparent"></span>
				{:else}
					<Zap class="h-4 w-4" />
				{/if}
				Quick silence (1 h)
			</button>

			<div class="h-4 w-px bg-border"></div>

			<!-- Customize actions: opens SilenceForm / AckForm pre-filled -->
			<button
				onclick={() => onBulkAck?.(computeCommonMatchers())}
				disabled={quickLoading}
				class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm bg-muted text-muted-foreground hover:bg-muted/70 disabled:opacity-50 transition-colors"
				title="Customize and bulk ack selected alerts"
			>
				<User class="h-4 w-4" />
				Customize ack…
			</button>

			<button
				onclick={() => onBulkSilence?.(computeCommonMatchers())}
				disabled={quickLoading}
				class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm bg-muted text-muted-foreground hover:bg-muted/70 disabled:opacity-50 transition-colors"
				title="Customize and bulk silence selected alerts"
			>
				<Volume2 class="h-4 w-4" />
				Customize silence…
			</button>
		{/if}

		<button
			onclick={clearSelection}
			class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
			title="Clear selection"
		>
			<X class="h-4 w-4" />
		</button>
	</div>
{/if}
