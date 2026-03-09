<script lang="ts">
	import { selectedFingerprints, alerts } from '$lib/stores/alerts';
	import { canSilence } from '$lib/stores/auth';
	import { X, Volume2, User } from 'lucide-svelte';
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

	// Compute the intersection of label matchers common to all selected alerts.
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
</script>

{#if count > 0}
	<div class="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 flex items-center gap-3 px-4 py-3 rounded-xl shadow-lg border bg-background">
		<span class="text-sm font-medium">{count} alert{count > 1 ? 's' : ''} selected</span>
		<div class="h-4 w-px bg-border"></div>
		{#if $canSilence}
			<button
				onclick={() => onBulkAck?.(computeCommonMatchers())}
				class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm bg-purple-100 text-purple-800 hover:bg-purple-200 transition-colors"
			>
				<User class="h-4 w-4" />
				Bulk ack
			</button>
			<button
				onclick={() => onBulkSilence?.(computeCommonMatchers())}
				class="flex items-center gap-1.5 px-3 py-1.5 rounded-md text-sm bg-orange-100 text-orange-800 hover:bg-orange-200 transition-colors"
			>
				<Volume2 class="h-4 w-4" />
				Bulk silence
			</button>
		{/if}
		<button
			onclick={clearSelection}
			class="p-1.5 rounded-md text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
		>
			<X class="h-4 w-4" />
		</button>
	</div>
{/if}
