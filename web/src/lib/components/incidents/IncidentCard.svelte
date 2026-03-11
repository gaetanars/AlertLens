<!--
  IncidentCard — compact summary card for use in the incidents list / dashboard.
  Clicking the card navigates to the detail view. Action buttons allow
  quick lifecycle transitions without opening the full detail page.
-->
<script lang="ts">
	import type { IncidentListItem } from '$lib/api/types';
	import IncidentStatusBadge from './IncidentStatusBadge.svelte';
	import { formatRelative } from '$lib/utils/duration';
	import { getSeverity, SEVERITY_BADGE } from '$lib/utils/severity';
	import { isAdmin } from '$lib/stores/auth';
	import { dispatchEvent } from '$lib/stores/incidents';
	import { AlertTriangle, Clock, Hash, User, MessageSquare } from 'lucide-svelte';
	import { get } from 'svelte/store';

	let {
		incident,
		onClick,
		compact = false
	}: {
		incident: IncidentListItem;
		onClick?: (incident: IncidentListItem) => void;
		compact?: boolean;
	} = $props();

	// Derive severity badge class using the shared utility.
	const severityClass = $derived(
		SEVERITY_BADGE[getSeverity({ severity: incident.severity })] ??
			'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300'
	);

	// Whether the current user can take lifecycle actions.
	const canAct = $derived(get(isAdmin));

	let actingAs = $state<'ACK' | 'INVESTIGATING' | 'RESOLVED' | 'REOPENED' | null>(null);

	async function quickTransition(kind: 'ACK' | 'INVESTIGATING' | 'RESOLVED' | 'REOPENED') {
		if (actingAs) return;
		actingAs = kind;
		try {
			await dispatchEvent(incident.id, { kind, actor: 'admin' });
		} finally {
			actingAs = null;
		}
	}

	// Which quick actions are available depends on current status.
	const quickActions = $derived(
		(() => {
			const s = incident.status;
			const actions: Array<{
				label: string;
				kind: 'ACK' | 'INVESTIGATING' | 'RESOLVED' | 'REOPENED';
				classes: string;
			}> = [];

			if (s === 'OPEN') {
				actions.push({
					label: 'Ack',
					kind: 'ACK',
					classes:
						'bg-yellow-100 text-yellow-800 hover:bg-yellow-200 dark:bg-yellow-900/30 dark:text-yellow-300'
				});
				actions.push({
					label: 'Investigate',
					kind: 'INVESTIGATING',
					classes:
						'bg-blue-100 text-blue-800 hover:bg-blue-200 dark:bg-blue-900/30 dark:text-blue-300'
				});
			}
			if (s === 'ACK') {
				actions.push({
					label: 'Investigate',
					kind: 'INVESTIGATING',
					classes:
						'bg-blue-100 text-blue-800 hover:bg-blue-200 dark:bg-blue-900/30 dark:text-blue-300'
				});
			}
			if (s === 'ACK' || s === 'INVESTIGATING') {
				actions.push({
					label: 'Resolve',
					kind: 'RESOLVED',
					classes:
						'bg-green-100 text-green-800 hover:bg-green-200 dark:bg-green-900/30 dark:text-green-300'
				});
			}
			if (s === 'RESOLVED') {
				actions.push({
					label: 'Reopen',
					kind: 'REOPENED',
					classes:
						'bg-orange-100 text-orange-800 hover:bg-orange-200 dark:bg-orange-900/30 dark:text-orange-300'
				});
			}
			return actions;
		})()
	);
</script>

<article
	class="rounded-lg border bg-card shadow-sm cursor-pointer transition-all hover:shadow-md
		hover:border-primary/40 focus-within:ring-2 focus-within:ring-ring
		{compact ? 'p-3' : 'p-4'}"
	onclick={() => onClick?.(incident)}
	onkeydown={(e) => e.key === 'Enter' && onClick?.(incident)}
	role="button"
	tabindex="0"
	aria-label="Incident: {incident.title}"
>
	<!-- Header -->
	<div class="flex items-start justify-between gap-2 mb-2">
		<div class="flex items-start gap-2 min-w-0">
			<AlertTriangle class="h-4 w-4 mt-0.5 flex-shrink-0 text-muted-foreground" />
			<div class="min-w-0">
				<h3 class="text-sm font-semibold leading-tight truncate" title={incident.title}>
					{incident.title}
				</h3>
				{#if !compact && incident.alertmanager_instance}
					<p class="text-xs text-muted-foreground mt-0.5">
						{incident.alertmanager_instance}
					</p>
				{/if}
			</div>
		</div>
		<div class="flex items-center gap-1.5 flex-shrink-0">
			<span class="text-xs px-1.5 py-0.5 rounded-full {severityClass}">
				{incident.severity}
			</span>
			<IncidentStatusBadge status={incident.status} />
		</div>
	</div>

	<!-- Labels -->
	{#if !compact && incident.labels && Object.keys(incident.labels).length > 0}
		<div class="flex flex-wrap gap-1 mb-2">
			{#each Object.entries(incident.labels).slice(0, 4) as [k, v]}
				<span class="text-xs px-1 py-0.5 rounded bg-muted text-muted-foreground">
					{k}=<strong>{v}</strong>
				</span>
			{/each}
			{#if Object.keys(incident.labels).length > 4}
				<span class="text-xs text-muted-foreground">+{Object.keys(incident.labels).length - 4} more</span>
			{/if}
		</div>
	{/if}

	<!-- Footer row -->
	<div class="flex items-center justify-between gap-2 mt-2">
		<div class="flex items-center gap-3 text-xs text-muted-foreground">
			<span class="flex items-center gap-1">
				<Clock class="h-3 w-3" />
				{formatRelative(incident.created_at)}
			</span>
			<span class="flex items-center gap-1">
				<MessageSquare class="h-3 w-3" />
				{incident.event_count} event{incident.event_count !== 1 ? 's' : ''}
			</span>
			{#if incident.alert_fingerprint}
				<span class="flex items-center gap-1 font-mono">
					<Hash class="h-3 w-3" />
					{incident.alert_fingerprint.slice(0, 8)}
				</span>
			{/if}
		</div>

		<!-- Quick actions -->
		{#if canAct && quickActions.length > 0}
			<div
				class="flex gap-1"
				onclick={(e) => e.stopPropagation()}
				role="presentation"
			>
				{#each quickActions as action}
					<button
						onclick={() => quickTransition(action.kind)}
						disabled={actingAs !== null}
						class="text-xs px-2 py-0.5 rounded transition-colors disabled:opacity-50
							{action.classes}"
					>
						{actingAs === action.kind ? '…' : action.label}
					</button>
				{/each}
			</div>
		{/if}
	</div>
</article>
