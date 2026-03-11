<!--
  IncidentTimeline — visualises the immutable event ledger for a single incident.

  Each event is rendered as a node on a vertical timeline, colour-coded by kind.
  CREATED and status-changing events show a filled circle; COMMENT events show a
  hollow ring. The connector line is continuous between events.
-->
<script lang="ts">
	import type { IncidentEvent, IncidentEventKind } from '$lib/api/types';
	import { formatRelative, formatDuration } from '$lib/utils/duration';
	import {
		AlertCircle,
		CheckCircle2,
		Search,
		CheckCheck,
		RefreshCw,
		MessageSquare,
		Zap
	} from 'lucide-svelte';

	let { events, class: className = '' }: { events: IncidentEvent[]; class?: string } = $props();

	// ─── Event metadata ───────────────────────────────────────────────────────

	interface EventMeta {
		icon: typeof AlertCircle;
		color: string;
		connectorColor: string;
		bgColor: string;
		borderColor: string;
		label: string;
		hollow?: boolean;
	}

	const META: Record<IncidentEventKind, EventMeta> = {
		CREATED: {
			icon: Zap,
			color: 'text-red-600 dark:text-red-400',
			connectorColor: 'bg-red-300 dark:bg-red-700',
			bgColor: 'bg-red-50 dark:bg-red-950/40',
			borderColor: 'border-red-200 dark:border-red-800',
			label: 'Incident opened'
		},
		ACK: {
			icon: CheckCircle2,
			color: 'text-yellow-600 dark:text-yellow-400',
			connectorColor: 'bg-yellow-300 dark:bg-yellow-700',
			bgColor: 'bg-yellow-50 dark:bg-yellow-950/40',
			borderColor: 'border-yellow-200 dark:border-yellow-800',
			label: 'Acknowledged'
		},
		INVESTIGATING: {
			icon: Search,
			color: 'text-blue-600 dark:text-blue-400',
			connectorColor: 'bg-blue-300 dark:bg-blue-700',
			bgColor: 'bg-blue-50 dark:bg-blue-950/40',
			borderColor: 'border-blue-200 dark:border-blue-800',
			label: 'Investigation started'
		},
		RESOLVED: {
			icon: CheckCheck,
			color: 'text-green-600 dark:text-green-400',
			connectorColor: 'bg-green-300 dark:bg-green-700',
			bgColor: 'bg-green-50 dark:bg-green-950/40',
			borderColor: 'border-green-200 dark:border-green-800',
			label: 'Resolved'
		},
		REOPENED: {
			icon: RefreshCw,
			color: 'text-orange-600 dark:text-orange-400',
			connectorColor: 'bg-orange-300 dark:bg-orange-700',
			bgColor: 'bg-orange-50 dark:bg-orange-950/40',
			borderColor: 'border-orange-200 dark:border-orange-800',
			label: 'Reopened'
		},
		COMMENT: {
			icon: MessageSquare,
			color: 'text-gray-500 dark:text-gray-400',
			connectorColor: 'bg-gray-200 dark:bg-gray-700',
			bgColor: 'bg-muted/40',
			borderColor: 'border-border',
			label: 'Comment',
			hollow: true
		}
	};

	const fallbackMeta: EventMeta = {
		icon: AlertCircle,
		color: 'text-muted-foreground',
		connectorColor: 'bg-border',
		bgColor: 'bg-muted/20',
		borderColor: 'border-border',
		label: 'Event'
	};

	function getMeta(kind: IncidentEventKind): EventMeta {
		return META[kind] ?? fallbackMeta;
	}

	// Duration between consecutive events.
	function durationBetween(prev: IncidentEvent, curr: IncidentEvent): string {
		const ms =
			new Date(curr.occurred_at).getTime() - new Date(prev.occurred_at).getTime();
		if (ms < 1000) return '';
		return formatDuration(ms);
	}
</script>

<ol class="relative {className}" aria-label="Incident timeline">
	{#each events as evt, i (evt.seq)}
		{@const meta = getMeta(evt.kind)}
		{@const isLast = i === events.length - 1}
		{@const IconComponent = meta.icon}

		<li class="relative flex gap-3 {isLast ? '' : 'pb-6'}">
			<!-- Connector line (not shown after last event) -->
			{#if !isLast}
				<div
					class="absolute left-4 top-8 w-0.5 h-full {meta.connectorColor}"
					aria-hidden="true"
				></div>
			{/if}

			<!-- Icon node -->
			<div
				class="relative z-10 flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full
					{meta.hollow
						? 'border-2 ' + meta.borderColor + ' bg-background'
						: meta.bgColor + ' border ' + meta.borderColor}"
			>
				<IconComponent class="h-4 w-4 {meta.color}" aria-hidden="true" />
			</div>

			<!-- Content card -->
			<div class="flex-1 min-w-0">
				<!-- Header row -->
				<div class="flex items-center justify-between gap-2 flex-wrap">
					<div class="flex items-center gap-2">
						<span class="text-sm font-medium {meta.color}">
							{meta.label}
						</span>
						{#if evt.status}
							<span
								class="text-xs px-1.5 py-0.5 rounded-full font-mono
									{meta.bgColor} {meta.color} border {meta.borderColor}"
							>
								{evt.status}
							</span>
						{/if}
					</div>
					<div class="flex items-center gap-2 flex-shrink-0">
						<!-- Duration since previous event -->
						{#if i > 0}
							{@const dur = durationBetween(events[i - 1], evt)}
							{#if dur}
								<span class="text-xs text-muted-foreground" title="Time since previous event">
									+{dur}
								</span>
							{/if}
						{/if}
						<time
							class="text-xs text-muted-foreground"
							datetime={evt.occurred_at}
							title={new Date(evt.occurred_at).toLocaleString()}
						>
							{formatRelative(evt.occurred_at)}
						</time>
					</div>
				</div>

				<!-- Actor -->
				<p class="text-xs text-muted-foreground mt-0.5">
					by <span class="font-medium text-foreground">{evt.actor}</span>
				</p>

				<!-- Message / annotation -->
				{#if evt.message}
					<blockquote
						class="mt-1.5 text-sm text-foreground/80 italic border-l-2 {meta.borderColor} pl-3 py-0.5"
					>
						{evt.message}
					</blockquote>
				{/if}
			</div>
		</li>
	{/each}

	{#if events.length === 0}
		<li class="text-sm text-muted-foreground italic">No events recorded.</li>
	{/if}
</ol>
