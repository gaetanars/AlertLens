<script lang="ts">
	import { onMount } from 'svelte';
	import { diffConfig, saveConfig } from '$lib/api/config';
	import {
		listTimeIntervals,
		deleteTimeInterval,
		validateTimeInterval,
		exportConfig
	} from '$lib/api/builder';
	import type { TimeIntervalEntry, TimeIntervalDef, TimeRangeDef } from '$lib/api/types';
	import YamlDiffViewer from '$lib/components/config/YamlDiffViewer.svelte';
	import { instances } from '$lib/stores/alerts';
	import { canEditConfig } from '$lib/stores/auth';
	import { configDraftStore } from '$lib/stores/configDraft';
	import { toast } from 'svelte-sonner';
	import { Plus, Trash2, Save, Eye, Lock } from 'lucide-svelte';

	const WEEKDAYS = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'];

	function emptySpec(): TimeIntervalDef {
		return { times: [], weekdays: [], days_of_month: [], months: [], years: [], location: '' };
	}

	let selectedInstance = $state($instances[0]?.name ?? '');
	let intervals = $state<TimeIntervalEntry[]>([]);
	let editingIdx = $state<number | null>(null);
	let loading = $state(false);
	let saving = $state(false);
	let validationErrors = $state<string[]>([]);

	// Diff/save step
	let step = $state<'edit' | 'diff'>('edit');
	let diffResult = $state<{ diff: string; has_changes: boolean } | null>(null);
	let pendingYaml = $state('');

	// Save options
	let saveMode = $state<'disk' | 'github' | 'gitlab'>('disk');
	let diskPath = $state('');
	let gitRepo = $state('');
	let gitBranch = $state('main');
	let gitFilePath = $state('alertmanager.yml');
	let webhookUrl = $state('');

	// Debounced validation for the selected interval
	let validationTimer: ReturnType<typeof setTimeout> | null = null;
	$effect(() => {
		const idx = editingIdx;
		if (idx === null || !$canEditConfig) return;
		const entry = intervals[idx];
		if (!entry) return;
		if (validationTimer) clearTimeout(validationTimer);
		validationTimer = setTimeout(async () => {
			try {
				const result = await validateTimeInterval(entry);
				validationErrors = result.valid ? [] : (result.errors ?? []);
			} catch {
				// ignore transient errors
			}
		}, 500);
	});

	async function load() {
		loading = true;
		try {
			const resp = await listTimeIntervals(selectedInstance || undefined);
			intervals = resp.time_intervals;
		} catch (e) {
			toast.error('Load error: ' + (e instanceof Error ? e.message : ''));
		} finally {
			loading = false;
		}
	}

	function patchInterval(idx: number, patch: Partial<TimeIntervalEntry>) {
		intervals = intervals.map((iv, i) => i === idx ? { ...iv, ...patch } : iv);
	}

	function patchSpec(intervalIdx: number, specIdx: number, patch: Partial<TimeIntervalDef>) {
		const specs = intervals[intervalIdx].time_intervals.map((s, i) =>
			i === specIdx ? { ...s, ...patch } : s
		);
		patchInterval(intervalIdx, { time_intervals: specs });
	}

	function addInterval() {
		intervals = [...intervals, { name: '', time_intervals: [emptySpec()] }];
		editingIdx = intervals.length - 1;
		validationErrors = [];
	}

	async function removeInterval(idx: number) {
		const name = intervals[idx].name;
		if (!name) {
			// Draft not yet saved — just remove locally
			intervals = intervals.filter((_, i) => i !== idx);
			if (editingIdx === idx) editingIdx = null;
			return;
		}
		try {
			const result = await deleteTimeInterval(name, selectedInstance || undefined);
			toast.success(`Time interval "${name}" deleted`);
			if (editingIdx === idx) editingIdx = null;
			// Share the assembled YAML with the Save & Deploy tab.
			configDraftStore.set({ instance: selectedInstance, rawYaml: result.raw_yaml });
			await load();
		} catch (e) {
			toast.error('Delete error: ' + (e instanceof Error ? e.message : ''));
		}
	}

	function addSpec(intervalIdx: number) {
		const specs = [...intervals[intervalIdx].time_intervals, emptySpec()];
		patchInterval(intervalIdx, { time_intervals: specs });
	}

	function removeSpec(intervalIdx: number, specIdx: number) {
		const specs = intervals[intervalIdx].time_intervals.filter((_, i) => i !== specIdx);
		patchInterval(intervalIdx, { time_intervals: specs });
	}

	// Time ranges
	function addTimeRange(intervalIdx: number, specIdx: number) {
		const spec = intervals[intervalIdx].time_intervals[specIdx];
		patchSpec(intervalIdx, specIdx, { times: [...(spec.times ?? []), { start_time: '', end_time: '' }] });
	}

	function removeTimeRange(intervalIdx: number, specIdx: number, rangeIdx: number) {
		const spec = intervals[intervalIdx].time_intervals[specIdx];
		patchSpec(intervalIdx, specIdx, { times: (spec.times ?? []).filter((_, i) => i !== rangeIdx) });
	}

	function patchTimeRange(intervalIdx: number, specIdx: number, rangeIdx: number, patch: Partial<TimeRangeDef>) {
		const spec = intervals[intervalIdx].time_intervals[specIdx];
		const times = (spec.times ?? []).map((t, i) => i === rangeIdx ? { ...t, ...patch } : t);
		patchSpec(intervalIdx, specIdx, { times });
	}

	// Weekday toggle: only toggle single-day names, leave range syntax (e.g. "monday:friday") untouched
	function toggleWeekday(intervalIdx: number, specIdx: number, day: string) {
		const spec = intervals[intervalIdx].time_intervals[specIdx];
		const current = spec.weekdays ?? [];
		const isSingleDay = (s: string) => WEEKDAYS.includes(s);
		if (current.includes(day)) {
			patchSpec(intervalIdx, specIdx, { weekdays: current.filter(d => d !== day) });
		} else {
			patchSpec(intervalIdx, specIdx, { weekdays: [...current, day] });
		}
		// Range-syntax entries (e.g. "monday:friday") are preserved unchanged by the filter above
		void isSingleDay; // reference to suppress unused warning
	}

	function isDayChecked(spec: TimeIntervalDef, day: string): boolean {
		return (spec.weekdays ?? []).includes(day);
	}

	// Text ↔ string[] helpers for days_of_month / months / years
	function arrToText(arr?: string[]): string {
		return (arr ?? []).join(', ');
	}

	function textToArr(text: string): string[] {
		return text.split(',').map(s => s.trim()).filter(Boolean);
	}

	async function previewDiff(idx: number) {
		const entry = intervals[idx];
		try {
			const exported = await exportConfig({
				instance: selectedInstance || undefined,
				time_intervals: [entry]
			});
			pendingYaml = exported.raw_yaml;
			diffResult = await diffConfig(selectedInstance || '', exported.raw_yaml);
			step = 'diff';
			// Share the assembled YAML with the Save & Deploy tab.
			configDraftStore.set({ instance: selectedInstance, rawYaml: exported.raw_yaml });
		} catch (e) {
			toast.error('Diff error: ' + (e instanceof Error ? e.message : ''));
		}
	}

	async function save() {
		saving = true;
		try {
			await saveConfig({
				alertmanager: selectedInstance || '',
				raw_yaml: pendingYaml,
				save_mode: saveMode,
				disk_options: saveMode === 'disk' ? { file_path: diskPath } : undefined,
				git_options: saveMode !== 'disk' ? {
					repo: gitRepo, branch: gitBranch, file_path: gitFilePath
				} : undefined,
				webhook_url: webhookUrl || undefined
			});
			toast.success('Configuration saved successfully');
			await load();
			step = 'edit';
			diffResult = null;
			pendingYaml = '';
		} catch (e) {
			toast.error('Save error: ' + (e instanceof Error ? e.message : ''));
		} finally {
			saving = false;
		}
	}

	onMount(() => { load(); });
</script>

<div class="mt-4 space-y-4">
	<div class="flex items-center justify-between">
		<div>
			<h2 class="font-semibold">Time Intervals</h2>
			<p class="text-xs text-muted-foreground mt-0.5">
				Named time intervals usable in routes as <code class="bg-muted px-1 rounded">mute_time_intervals</code> or <code class="bg-muted px-1 rounded">active_time_intervals</code>.
			</p>
		</div>
		<select bind:value={selectedInstance} onchange={load} class="px-2 py-1 rounded border bg-background text-sm">
			<option value="">Default</option>
			{#each $instances as inst}
				<option value={inst.name}>{inst.name}</option>
			{/each}
		</select>
	</div>

	{#if !$canEditConfig}
		<div class="flex items-center gap-2 p-3 rounded-lg border bg-muted/30 text-sm text-muted-foreground">
			<Lock class="h-4 w-4 shrink-0" />
			You need the <strong class="mx-1 text-foreground">config-editor</strong> role to edit time intervals.
		</div>
	{/if}

	{#if loading}
		<div class="py-8 text-center text-muted-foreground animate-pulse">Loading…</div>
	{:else if step === 'edit'}
		<div class="space-y-3">
			{#each intervals as interval, i}
				<div class="rounded-lg border bg-card overflow-hidden">
					<!-- Interval header -->
					<div class="flex items-center gap-2 px-3 py-2 bg-muted/30 border-b">
						{#if $canEditConfig}
							<input
								value={interval.name}
								oninput={(e) => patchInterval(i, { name: (e.target as HTMLInputElement).value })}
								placeholder="interval-name"
								class="flex-1 px-2 py-1 rounded border bg-background text-sm font-medium"
							/>
						{:else}
							<span class="flex-1 text-sm font-medium px-2">{interval.name}</span>
						{/if}
						<button
							onclick={() => editingIdx = editingIdx === i ? null : i}
							class="text-xs text-muted-foreground hover:text-foreground transition-colors px-2 py-1 rounded border"
						>
							{editingIdx === i ? 'Collapse' : 'Edit'}
						</button>
						{#if $canEditConfig}
							<button
								onclick={() => removeInterval(i)}
								class="p-1 rounded text-muted-foreground hover:text-destructive transition-colors"
								title="Delete interval"
							>
								<Trash2 class="h-3.5 w-3.5" />
							</button>
						{/if}
					</div>

					{#if editingIdx === i}
						<div class="p-3 space-y-3">
							{#each interval.time_intervals as spec, j}
								<div class="pl-3 border-l-2 border-muted space-y-2">
									<div class="flex items-center justify-between">
										<span class="text-xs font-medium text-muted-foreground">Spec {j + 1}</span>
										{#if $canEditConfig && interval.time_intervals.length > 1}
											<button
												onclick={() => removeSpec(i, j)}
												class="p-0.5 rounded text-muted-foreground hover:text-destructive transition-colors"
												title="Remove spec"
											>
												<Trash2 class="h-3 w-3" />
											</button>
										{/if}
									</div>

									<!-- Time ranges -->
									<div>
										<div class="flex items-center justify-between mb-1">
											<span class="text-xs text-muted-foreground">Time ranges</span>
											{#if $canEditConfig}
												<button
													onclick={() => addTimeRange(i, j)}
													class="flex items-center gap-1 text-xs text-primary hover:underline"
												>
													<Plus class="h-3 w-3" /> Add
												</button>
											{/if}
										</div>
										<div class="space-y-1">
											{#each spec.times ?? [] as t, ti}
												<div class="flex items-center gap-2">
													<input
														type="time"
														value={t.start_time}
														oninput={(e) => patchTimeRange(i, j, ti, { start_time: (e.target as HTMLInputElement).value })}
														disabled={!$canEditConfig}
														class="px-2 py-1 rounded border bg-background text-xs disabled:opacity-60"
													/>
													<span class="text-xs text-muted-foreground">–</span>
													<input
														type="time"
														value={t.end_time}
														oninput={(e) => patchTimeRange(i, j, ti, { end_time: (e.target as HTMLInputElement).value })}
														disabled={!$canEditConfig}
														class="px-2 py-1 rounded border bg-background text-xs disabled:opacity-60"
													/>
													{#if $canEditConfig}
														<button
															onclick={() => removeTimeRange(i, j, ti)}
															class="p-0.5 rounded text-muted-foreground hover:text-destructive transition-colors"
															title="Remove range"
														>
															<Trash2 class="h-3 w-3" />
														</button>
													{/if}
												</div>
											{/each}
											{#if (spec.times ?? []).length === 0}
												<p class="text-xs text-muted-foreground italic">No time ranges — all day</p>
											{/if}
										</div>
									</div>

									<!-- Weekdays -->
									<div>
										<span class="text-xs text-muted-foreground block mb-1">Days of week</span>
										<div class="flex flex-wrap gap-2">
											{#each WEEKDAYS as day}
												<label class="flex items-center gap-1 text-xs cursor-pointer">
													<input
														type="checkbox"
														checked={isDayChecked(spec, day)}
														onchange={() => $canEditConfig && toggleWeekday(i, j, day)}
														disabled={!$canEditConfig}
													/>
													{day.slice(0, 3)}
												</label>
											{/each}
										</div>
										{#if (spec.weekdays ?? []).some(d => d.includes(':'))}
											<p class="text-xs text-muted-foreground mt-1 italic">
												Range syntax preserved: {(spec.weekdays ?? []).filter(d => d.includes(':')).join(', ')}
											</p>
										{/if}
									</div>

									<!-- Days of month -->
									<div>
										<span class="text-xs text-muted-foreground block mb-1">Days of month</span>
										<input
											value={arrToText(spec.days_of_month)}
											oninput={(e) => patchSpec(i, j, { days_of_month: textToArr((e.target as HTMLInputElement).value) })}
											placeholder="1:15, -1"
											disabled={!$canEditConfig}
											class="w-full px-2 py-1 rounded border bg-background text-xs disabled:opacity-60"
										/>
									</div>

									<!-- Months -->
									<div>
										<span class="text-xs text-muted-foreground block mb-1">Months</span>
										<input
											value={arrToText(spec.months)}
											oninput={(e) => patchSpec(i, j, { months: textToArr((e.target as HTMLInputElement).value) })}
											placeholder="january:march, 12"
											disabled={!$canEditConfig}
											class="w-full px-2 py-1 rounded border bg-background text-xs disabled:opacity-60"
										/>
									</div>

									<!-- Years -->
									<div>
										<span class="text-xs text-muted-foreground block mb-1">Years</span>
										<input
											value={arrToText(spec.years)}
											oninput={(e) => patchSpec(i, j, { years: textToArr((e.target as HTMLInputElement).value) })}
											placeholder="2024:2026"
											disabled={!$canEditConfig}
											class="w-full px-2 py-1 rounded border bg-background text-xs disabled:opacity-60"
										/>
									</div>

									<!-- Timezone -->
									<div>
										<span class="text-xs text-muted-foreground block mb-1">Timezone</span>
										<input
											value={spec.location ?? ''}
											oninput={(e) => patchSpec(i, j, { location: (e.target as HTMLInputElement).value || undefined })}
											placeholder="Europe/Paris"
											disabled={!$canEditConfig}
											class="w-48 px-2 py-1 rounded border bg-background text-xs disabled:opacity-60"
										/>
									</div>
								</div>
							{/each}

							{#if $canEditConfig}
								<button
									onclick={() => addSpec(i)}
									class="flex items-center gap-1 text-xs text-primary hover:underline"
								>
									<Plus class="h-3 w-3" /> Add spec
								</button>
							{/if}

							<!-- Validation errors -->
							{#if validationErrors.length > 0 && editingIdx === i}
								<div class="rounded-md border border-destructive/50 bg-destructive/10 p-2 space-y-1">
									{#each validationErrors as err}
										<p class="text-xs text-destructive">{err}</p>
									{/each}
								</div>
							{/if}

							{#if $canEditConfig}
								<div class="flex gap-2 flex-wrap pt-1">
									<button
										onclick={() => previewDiff(i)}
										class="flex items-center gap-2 px-3 py-1.5 rounded-md border text-sm hover:bg-muted transition-colors"
									>
										<Eye class="h-4 w-4" />
										Preview diff
									</button>
								</div>
							{/if}
						</div>
					{/if}
				</div>
			{/each}

			{#if $canEditConfig}
				<button
					onclick={addInterval}
					class="flex items-center gap-2 px-4 py-2 rounded-md border border-dashed text-sm text-muted-foreground hover:text-foreground hover:border-foreground transition-colors"
				>
					<Plus class="h-4 w-4" />
					Add time interval
				</button>
			{/if}
		</div>

	{:else if step === 'diff' && diffResult}
		<YamlDiffViewer diff={diffResult.diff} hasChanges={diffResult.has_changes} />

		{#if diffResult.has_changes}
			<div class="p-4 rounded-lg border space-y-3 mt-3">
				<h3 class="font-semibold text-sm">Save mode</h3>
				<div class="flex gap-3">
					{#each ['disk', 'github', 'gitlab'] as mode}
						<label class="flex items-center gap-1.5 cursor-pointer">
							<input type="radio" bind:group={saveMode} value={mode} />
							<span class="text-sm capitalize">{mode}</span>
						</label>
					{/each}
				</div>
				{#if saveMode === 'disk'}
					<input bind:value={diskPath} placeholder="/etc/alertmanager/alertmanager.yml" class="w-full px-3 py-2 rounded border bg-background text-sm" />
				{:else}
					<input bind:value={gitRepo} placeholder="owner/repo" class="w-full px-3 py-2 rounded border bg-background text-sm mb-1" />
					<div class="grid grid-cols-2 gap-2">
						<input bind:value={gitBranch} placeholder="main" class="px-3 py-2 rounded border bg-background text-sm" />
						<input bind:value={gitFilePath} placeholder="alertmanager.yml" class="px-3 py-2 rounded border bg-background text-sm" />
					</div>
				{/if}
				<input bind:value={webhookUrl} placeholder="Webhook URL (optional)" class="w-full px-3 py-2 rounded border bg-background text-sm" />
			</div>
		{/if}

		<div class="flex gap-2 mt-3">
			<button onclick={() => step = 'edit'} class="px-4 py-2 rounded-md border text-sm hover:bg-muted transition-colors">Back</button>
			{#if diffResult.has_changes}
				<button
					onclick={save}
					disabled={saving}
					class="flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm disabled:opacity-50 transition-colors"
				>
					<Save class="h-4 w-4" />
					{saving ? 'Saving…' : 'Confirm and save'}
				</button>
			{/if}
		</div>
	{/if}
</div>
