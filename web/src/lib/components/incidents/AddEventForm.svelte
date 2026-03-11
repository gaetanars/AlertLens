<!--
  AddEventForm — modal/inline form for adding a lifecycle event or comment to
  an incident. Enforces the state machine by only showing valid transitions.
-->
<script lang="ts">
	import type { IncidentStatus, IncidentEventKind } from '$lib/api/types';
	import { dispatchEvent } from '$lib/stores/incidents';

	let {
		incidentId,
		currentStatus,
		onDone,
		onCancel
	}: {
		incidentId: string;
		currentStatus: IncidentStatus;
		onDone?: () => void;
		onCancel?: () => void;
	} = $props();

	// Only event kinds that can be submitted by users (CREATED is internal).
	type SubmittableKind = Exclude<IncidentEventKind, 'CREATED'>;

	interface KindOption {
		kind: SubmittableKind;
		label: string;
		description: string;
		classes: string;
	}

	// Build list of available options based on current status.
	const availableOptions = $derived(
		(() => {
			const opts: KindOption[] = [];

			const canACK =
				currentStatus === 'OPEN' || currentStatus === 'INVESTIGATING';
			const canINV =
				currentStatus === 'OPEN' || currentStatus === 'ACK';
			const canRES =
				currentStatus === 'OPEN' ||
				currentStatus === 'ACK' ||
				currentStatus === 'INVESTIGATING';
			const canREO = currentStatus === 'RESOLVED';

			if (canACK) {
				opts.push({
					kind: 'ACK',
					label: '✋ Acknowledge',
					description: 'Take ownership of this incident.',
					classes:
						'border-yellow-300 bg-yellow-50 dark:bg-yellow-950/20 data-[selected]:ring-yellow-400'
				});
			}
			if (canINV) {
				opts.push({
					kind: 'INVESTIGATING',
					label: '🔍 Start Investigation',
					description: 'Actively diagnosing the root cause.',
					classes:
						'border-blue-300 bg-blue-50 dark:bg-blue-950/20 data-[selected]:ring-blue-400'
				});
			}
			if (canRES) {
				opts.push({
					kind: 'RESOLVED',
					label: '✅ Resolve',
					description: 'Mark the incident as resolved.',
					classes:
						'border-green-300 bg-green-50 dark:bg-green-950/20 data-[selected]:ring-green-400'
				});
			}
			if (canREO) {
				opts.push({
					kind: 'REOPENED',
					label: '🔄 Reopen',
					description: 'The issue has recurred or was not fully resolved.',
					classes:
						'border-orange-300 bg-orange-50 dark:bg-orange-950/20 data-[selected]:ring-orange-400'
				});
			}
			// COMMENT is always available.
			opts.push({
				kind: 'COMMENT',
				label: '💬 Add Comment',
				description: 'Annotate without changing status.',
				classes:
					'border-border bg-muted/20 data-[selected]:ring-ring'
			});

			return opts;
		})()
	);

	let selectedKind = $state<SubmittableKind>('COMMENT');
	let actor = $state('');
	let message = $state('');
	let submitting = $state(false);
	let error = $state<string | null>(null);

	// Auto-select first available non-COMMENT option on mount / status change.
	$effect(() => {
		const first = availableOptions.find((o) => o.kind !== 'COMMENT');
		if (first) selectedKind = first.kind;
	});

	async function handleSubmit(e: SubmitEvent) {
		e.preventDefault();
		if (!actor.trim()) {
			error = 'Actor name is required.';
			return;
		}
		if (selectedKind === 'COMMENT' && !message.trim()) {
			error = 'A message is required for comments.';
			return;
		}
		submitting = true;
		error = null;
		try {
			await dispatchEvent(incidentId, {
				kind: selectedKind,
				actor: actor.trim(),
				message: message.trim() || undefined
			});
			onDone?.();
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to add event.';
		} finally {
			submitting = false;
		}
	}
</script>

<form onsubmit={handleSubmit} class="space-y-4">
	<!-- Event kind selector -->
	<fieldset>
		<legend class="text-sm font-medium mb-2">Action</legend>
		<div class="grid gap-2 sm:grid-cols-2">
			{#each availableOptions as opt}
				<label
					class="relative flex cursor-pointer rounded-lg border p-3 transition-all
						hover:shadow-sm {opt.classes}
						{selectedKind === opt.kind ? 'ring-2' : ''}"
				>
					<input
						type="radio"
						name="kind"
						value={opt.kind}
						bind:group={selectedKind}
						class="sr-only"
					/>
					<div>
						<span class="block text-sm font-medium">{opt.label}</span>
						<span class="block text-xs text-muted-foreground mt-0.5">
							{opt.description}
						</span>
					</div>
				</label>
			{/each}
		</div>
	</fieldset>

	<!-- Actor -->
	<div>
		<label for="actor" class="block text-sm font-medium mb-1">
			Your name <span class="text-destructive">*</span>
		</label>
		<input
			id="actor"
			type="text"
			bind:value={actor}
			placeholder="e.g. alice or SRE-bot"
			required
			class="w-full rounded-md border bg-background px-3 py-2 text-sm
				focus:outline-none focus:ring-2 focus:ring-ring placeholder:text-muted-foreground"
		/>
	</div>

	<!-- Message / note -->
	<div>
		<label for="message" class="block text-sm font-medium mb-1">
			Note
			{#if selectedKind === 'COMMENT'}
				<span class="text-destructive">*</span>
			{/if}
		</label>
		<textarea
			id="message"
			bind:value={message}
			rows={3}
			placeholder="Add context, findings, or a resolution summary…"
			class="w-full rounded-md border bg-background px-3 py-2 text-sm resize-none
				focus:outline-none focus:ring-2 focus:ring-ring placeholder:text-muted-foreground"
		></textarea>
	</div>

	<!-- Error -->
	{#if error}
		<p class="text-sm text-destructive">{error}</p>
	{/if}

	<!-- Actions -->
	<div class="flex justify-end gap-2">
		{#if onCancel}
			<button
				type="button"
				onclick={onCancel}
				class="px-4 py-2 text-sm rounded-md border hover:bg-muted transition-colors"
			>
				Cancel
			</button>
		{/if}
		<button
			type="submit"
			disabled={submitting}
			class="px-4 py-2 text-sm rounded-md bg-primary text-primary-foreground
				hover:bg-primary/90 transition-colors disabled:opacity-50"
		>
			{submitting ? 'Saving…' : 'Save'}
		</button>
	</div>
</form>
