<script lang="ts">
	import type { Silence } from '$lib/api/types';
	import { expireSilence } from '$lib/api/silences';
	import { loadSilences } from '$lib/stores/silences';
	import { formatRelative } from '$lib/utils/duration';
	import { toast } from 'svelte-sonner';
	import { Trash2, User, Pencil } from 'lucide-svelte';
	import { isAdmin } from '$lib/stores/auth';

	let { silences, onEdit }: {
		silences: Silence[];
		onEdit?: (silence: Silence) => void;
	} = $props();

	const active = $derived(silences.filter(s => s.status.state === 'active'));
	const pending = $derived(silences.filter(s => s.status.state === 'pending'));
	const expired = $derived(silences.filter(s => s.status.state === 'expired'));

	async function expire(s: Silence) {
		try {
			await expireSilence(s.id, s.alertmanager);
			toast.success('Silence expiré');
			await loadSilences();
		} catch (e) {
			toast.error(e instanceof Error ? e.message : 'Erreur');
		}
	}

	function isAck(s: Silence) {
		return s.matchers.some(m => m.name === 'alertlens_ack_type');
	}

	// QUA-07: correct Alertmanager matcher syntax display.
	function matcherLabel(m: { name: string; value: string; isRegex: boolean; isEqual: boolean }): string {
		const op = m.isRegex ? (m.isEqual ? '=~' : '!~') : (m.isEqual ? '=' : '!=');
		return `${m.name}${op}"${m.value}"`;
	}
</script>

{#snippet silenceRow(s: Silence)}
	<div class="flex items-start justify-between p-3 rounded-lg border hover:bg-muted/30 transition-colors">
		<div class="space-y-1 min-w-0">
			<div class="flex items-center gap-2">
				{#if isAck(s)}
					<span class="flex items-center gap-1 px-2 py-0.5 rounded-full text-xs bg-purple-100 text-purple-800 dark:bg-purple-900 dark:text-purple-200">
						<User class="h-3 w-3" />
						Ack visuel
					</span>
				{:else}
					<span class="px-2 py-0.5 rounded-full text-xs bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200">
						Silence
					</span>
				{/if}
				<span class="text-xs text-muted-foreground">{s.alertmanager}</span>
				<span class="text-xs text-muted-foreground">par <strong>{s.createdBy}</strong></span>
			</div>
			<div class="flex flex-wrap gap-1">
				{#each s.matchers.filter(m => !m.name.startsWith('alertlens_')) as m}
					<code class="text-xs px-1 rounded bg-muted">{matcherLabel(m)}</code>
				{/each}
			</div>
			{#if s.comment}
				<p class="text-xs text-muted-foreground italic">"{s.comment}"</p>
			{/if}
			<p class="text-xs text-muted-foreground">Jusqu'au {new Date(s.endsAt).toLocaleString('fr-FR')}</p>
		</div>
		{#if $isAdmin && s.status.state === 'active'}
			<div class="flex gap-1 flex-shrink-0">
				{#if onEdit}
					<button
						onclick={() => onEdit(s)}
						class="p-1.5 rounded text-muted-foreground hover:text-foreground hover:bg-muted transition-colors"
						title="Modifier"
					>
						<Pencil class="h-4 w-4" />
					</button>
				{/if}
				<button
					onclick={() => expire(s)}
					class="p-1.5 rounded text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
					title="Expirer maintenant"
				>
					<Trash2 class="h-4 w-4" />
				</button>
			</div>
		{/if}
	</div>
{/snippet}

<div class="space-y-6">
	{#if active.length > 0}
		<section>
			<h3 class="font-semibold mb-2 text-sm text-muted-foreground uppercase tracking-wide">
				Actifs ({active.length})
			</h3>
			<div class="space-y-2">
				{#each active as s (s.id)}{@render silenceRow(s)}{/each}
			</div>
		</section>
	{/if}

	{#if pending.length > 0}
		<section>
			<h3 class="font-semibold mb-2 text-sm text-muted-foreground uppercase tracking-wide">
				En attente ({pending.length})
			</h3>
			<div class="space-y-2">
				{#each pending as s (s.id)}{@render silenceRow(s)}{/each}
			</div>
		</section>
	{/if}

	{#if expired.length > 0}
		<section>
			<h3 class="font-semibold mb-2 text-sm text-muted-foreground uppercase tracking-wide opacity-60">
				Expirés ({expired.length})
			</h3>
			<div class="space-y-2 opacity-60">
				{#each expired.slice(0, 10) as s (s.id)}{@render silenceRow(s)}{/each}
			</div>
		</section>
	{/if}

	{#if silences.length === 0}
		<div class="py-12 text-center text-muted-foreground">Aucun silence</div>
	{/if}
</div>
