<script module lang="ts">
	export interface FormMatcher {
		name: string;
		value: string;
		isRegex: boolean;
		isEqual: boolean;
	}

	export interface RouteFormNode {
		receiver: string;
		continue: boolean;
		group_by: string[];
		group_wait: string;
		group_interval: string;
		repeat_interval: string;
		matchers: FormMatcher[];
		mute_time_intervals: string[];
		active_time_intervals: string[];
		routes: RouteFormNode[];
	}

	export function emptyNode(): RouteFormNode {
		return {
			receiver: '',
			continue: false,
			group_by: [],
			group_wait: '',
			group_interval: '',
			repeat_interval: '',
			matchers: [],
			mute_time_intervals: [],
			active_time_intervals: [],
			routes: []
		};
	}
</script>

<script lang="ts">
	import { Plus, Trash2, ChevronDown, ChevronRight } from 'lucide-svelte';
	import RouteNodeEditor from './RouteNodeEditor.svelte';

	let { route, onUpdate, depth = 0, isRoot = false, availableTimeIntervals = [] }: {
		route: RouteFormNode;
		onUpdate: (r: RouteFormNode) => void;
		depth?: number;
		isRoot?: boolean;
		availableTimeIntervals?: string[];
	} = $props();

	let collapsed = $state(false);

	function update(patch: Partial<RouteFormNode>) {
		onUpdate({ ...route, ...patch });
	}

	function addMatcher() {
		update({ matchers: [...route.matchers, { name: '', value: '', isRegex: false, isEqual: true }] });
	}

	function removeMatcher(i: number) {
		update({ matchers: route.matchers.filter((_, idx) => idx !== i) });
	}

	function patchMatcher(i: number, patch: Partial<FormMatcher>) {
		const matchers = route.matchers.map((m, idx) => idx === i ? { ...m, ...patch } : m);
		update({ matchers });
	}

	function addChild() {
		update({ routes: [...route.routes, emptyNode()] });
	}

	function removeChild(i: number) {
		update({ routes: route.routes.filter((_, idx) => idx !== i) });
	}

	function patchChild(i: number, child: RouteFormNode) {
		const routes = route.routes.map((r, idx) => idx === i ? child : r);
		update({ routes });
	}

	function toggleGroupBy(label: string) {
		const cur = route.group_by;
		const next = cur.includes(label) ? cur.filter(l => l !== label) : [...cur, label];
		update({ group_by: next });
	}

	function toggleTimeInterval(field: 'mute_time_intervals' | 'active_time_intervals', name: string) {
		const cur = route[field];
		const next = cur.includes(name) ? cur.filter(n => n !== name) : [...cur, name];
		update({ [field]: next });
	}

	const COMMON_GROUP_BY = ['alertname', 'cluster', 'namespace', 'severity', 'job'];
</script>

<div class="rounded-lg border bg-card {depth > 0 ? 'ml-4 mt-2' : ''} overflow-hidden">
	<!-- Header -->
	<div class="flex items-center gap-2 px-3 py-2 bg-muted/30 border-b">
		<button onclick={() => collapsed = !collapsed} class="p-0.5 rounded hover:bg-muted transition-colors">
			{#if collapsed}
				<ChevronRight class="h-4 w-4" />
			{:else}
				<ChevronDown class="h-4 w-4" />
			{/if}
		</button>
		<span class="text-xs font-medium text-muted-foreground">{isRoot ? 'Root route' : `Route (depth ${depth})`}</span>
		<div class="flex-1">
			<input
				value={route.receiver}
				oninput={(e) => update({ receiver: (e.target as HTMLInputElement).value })}
				placeholder="receiver"
				class="w-full px-2 py-0.5 rounded border bg-background text-sm font-medium"
			/>
		</div>
		{#if !isRoot}
			<label class="flex items-center gap-1 text-xs cursor-pointer">
				<input
					type="checkbox"
					checked={route.continue}
					onchange={(e) => update({ continue: (e.target as HTMLInputElement).checked })}
				/>
				continue
			</label>
		{/if}
	</div>

	{#if !collapsed}
		<div class="p-3 space-y-3">
			<!-- Matchers -->
			<div>
				<div class="flex items-center justify-between mb-1">
					<span class="text-xs font-medium text-muted-foreground">Matchers</span>
					<button onclick={addMatcher} class="flex items-center gap-1 text-xs text-primary hover:underline">
						<Plus class="h-3 w-3" /> Add
					</button>
				</div>
				<div class="space-y-1">
					{#each route.matchers as m, i}
						<div class="flex gap-1 items-center">
							<input
								value={m.name}
								oninput={(e) => patchMatcher(i, { name: (e.target as HTMLInputElement).value })}
								placeholder="label"
								class="flex-1 px-2 py-1 rounded border bg-background text-xs"
							/>
							<select
								value={m.isEqual}
								onchange={(e) => patchMatcher(i, { isEqual: (e.target as HTMLSelectElement).value === 'true' })}
								class="px-1 py-1 rounded border bg-background text-xs"
							>
								<option value="true">=</option>
								<option value="false">!=</option>
							</select>
							<label class="flex items-center gap-1 text-xs cursor-pointer whitespace-nowrap">
								<input
									type="checkbox"
									checked={m.isRegex}
									onchange={(e) => patchMatcher(i, { isRegex: (e.target as HTMLInputElement).checked })}
								/>
								~
							</label>
							<input
								value={m.value}
								oninput={(e) => patchMatcher(i, { value: (e.target as HTMLInputElement).value })}
								placeholder="value"
								class="flex-1 px-2 py-1 rounded border bg-background text-xs"
							/>
							<button onclick={() => removeMatcher(i)} class="p-0.5 rounded text-muted-foreground hover:text-destructive transition-colors">
								<Trash2 class="h-3 w-3" />
							</button>
						</div>
					{/each}
				</div>
			</div>

			<!-- Group By -->
			<div>
				<span class="text-xs font-medium text-muted-foreground block mb-1">Group By</span>
				<div class="flex flex-wrap gap-1">
					{#each COMMON_GROUP_BY as label}
						<button
							onclick={() => toggleGroupBy(label)}
							class="px-2 py-0.5 rounded-full text-xs border transition-colors
								{route.group_by.includes(label) ? 'bg-primary text-primary-foreground border-primary' : 'hover:bg-muted'}"
						>
							{label}
						</button>
					{/each}
				</div>
			</div>

			<!-- Timing -->
			<div class="grid grid-cols-3 gap-2">
				<div>
					<span class="text-xs text-muted-foreground">group_wait</span>
					<input
						value={route.group_wait}
						oninput={(e) => update({ group_wait: (e.target as HTMLInputElement).value })}
						placeholder="30s"
						class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
					/>
				</div>
				<div>
					<span class="text-xs text-muted-foreground">group_interval</span>
					<input
						value={route.group_interval}
						oninput={(e) => update({ group_interval: (e.target as HTMLInputElement).value })}
						placeholder="5m"
						class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
					/>
				</div>
				<div>
					<span class="text-xs text-muted-foreground">repeat_interval</span>
					<input
						value={route.repeat_interval}
						oninput={(e) => update({ repeat_interval: (e.target as HTMLInputElement).value })}
						placeholder="4h"
						class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
					/>
				</div>
			</div>

			<!-- Time Intervals (not available on root route) -->
			{#if !isRoot}
				<div class="grid grid-cols-2 gap-3">
					<!-- Mute Time Intervals -->
					<div>
						<span class="text-xs font-medium text-muted-foreground block mb-1">Mute Time Intervals</span>
						{#if availableTimeIntervals.length === 0}
							<p class="text-xs text-muted-foreground italic">No interval defined — <a href="/config/time-intervals" class="text-primary hover:underline">create a Time Interval</a></p>
						{:else}
							<div class="flex flex-wrap gap-1">
								{#each availableTimeIntervals as name}
									<button
										onclick={() => toggleTimeInterval('mute_time_intervals', name)}
										class="px-2 py-0.5 rounded-full text-xs border transition-colors
											{route.mute_time_intervals.includes(name) ? 'bg-orange-500/20 text-orange-700 dark:text-orange-400 border-orange-500/50' : 'hover:bg-muted'}"
										title="Suppresses notifications during this interval"
									>
										{name}
									</button>
								{/each}
							</div>
						{/if}
					</div>

					<!-- Active Time Intervals -->
					<div>
						<span class="text-xs font-medium text-muted-foreground block mb-1">Active Time Intervals</span>
						{#if availableTimeIntervals.length === 0}
							<p class="text-xs text-muted-foreground italic">No interval defined — <a href="/config/time-intervals" class="text-primary hover:underline">create a Time Interval</a></p>
						{:else}
							<div class="flex flex-wrap gap-1">
								{#each availableTimeIntervals as name}
									<button
										onclick={() => toggleTimeInterval('active_time_intervals', name)}
										class="px-2 py-0.5 rounded-full text-xs border transition-colors
											{route.active_time_intervals.includes(name) ? 'bg-green-500/20 text-green-700 dark:text-green-400 border-green-500/50' : 'hover:bg-muted'}"
										title="Only sends notifications during this interval"
									>
										{name}
									</button>
								{/each}
							</div>
						{/if}
					</div>
				</div>
			{/if}

			<!-- Child routes -->
			{#if route.routes.length > 0}
				<div>
					<span class="text-xs font-medium text-muted-foreground block mb-1">Child routes ({route.routes.length})</span>
					{#each route.routes as child, i}
						<div class="relative">
							<RouteNodeEditor
								route={child}
								onUpdate={(updated) => patchChild(i, updated)}
								depth={depth + 1}
								availableTimeIntervals={availableTimeIntervals}
							/>
							<button
								onclick={() => removeChild(i)}
								class="absolute top-2 right-2 p-0.5 rounded text-muted-foreground hover:text-destructive transition-colors z-10"
								title="Remove this route"
							>
								<Trash2 class="h-3 w-3" />
							</button>
						</div>
					{/each}
				</div>
			{/if}

			<button onclick={addChild} class="flex items-center gap-1 text-xs text-primary hover:underline">
				<Plus class="h-3 w-3" /> Add child route
			</button>
		</div>
	{/if}
</div>
