<script lang="ts">
	import type { IncidentStatus } from '$lib/api/types';

	let { status }: { status: IncidentStatus } = $props();

	const CONFIG: Record<
		IncidentStatus,
		{ label: string; classes: string; dot: string }
	> = {
		OPEN: {
			label: 'Open',
			classes:
				'bg-red-100 text-red-800 dark:bg-red-900/40 dark:text-red-300 border border-red-200 dark:border-red-800',
			dot: 'bg-red-500 animate-pulse'
		},
		ACK: {
			label: 'Ack',
			classes:
				'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-300 border border-yellow-200 dark:border-yellow-800',
			dot: 'bg-yellow-500'
		},
		INVESTIGATING: {
			label: 'Investigating',
			classes:
				'bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300 border border-blue-200 dark:border-blue-800',
			dot: 'bg-blue-500 animate-pulse'
		},
		RESOLVED: {
			label: 'Resolved',
			classes:
				'bg-green-100 text-green-800 dark:bg-green-900/40 dark:text-green-300 border border-green-200 dark:border-green-800',
			dot: 'bg-green-500'
		}
	};

	const cfg = $derived(CONFIG[status] ?? CONFIG.OPEN);
</script>

<span
	class="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-xs font-medium {cfg.classes}"
>
	<span class="w-1.5 h-1.5 rounded-full {cfg.dot}"></span>
	{cfg.label}
</span>
