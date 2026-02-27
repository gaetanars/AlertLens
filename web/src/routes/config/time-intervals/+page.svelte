<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchConfig, validateConfig, diffConfig, saveConfig } from '$lib/api/config';
	import YamlDiffViewer from '$lib/components/config/YamlDiffViewer.svelte';
	import { instances } from '$lib/stores/alerts';
	import { toast } from 'svelte-sonner';
	import { Plus, Trash2, Save, Eye } from 'lucide-svelte';
	import * as yaml from 'js-yaml';

	let selectedInstance = $state($instances[0]?.name ?? '');
	let fullConfig = $state('');
	let timeIntervals = $state<TimeInterval[]>([]);
	let loading = $state(false);
	let saving = $state(false);
	let diffResult = $state<{ diff: string; has_changes: boolean } | null>(null);
	let step = $state<'edit' | 'diff'>('edit');

	// Save options
	let saveMode = $state<'disk' | 'github' | 'gitlab'>('disk');
	let diskPath = $state('');
	let gitRepo = $state('');
	let gitBranch = $state('main');
	let gitFilePath = $state('alertmanager.yml');
	let webhookUrl = $state('');
	let proposedYaml = $state('');

	interface TimeRange {
		start_time: string;
		end_time: string;
	}
	interface TimeIntervalSpec {
		weekdays?: string[];
		months?: string[];
		days_of_month?: string[];
		years?: string[];
		times?: TimeRange[];
		location?: string;
	}
	interface TimeInterval {
		name: string;
		time_intervals: TimeIntervalSpec[];
	}

	async function load() {
		loading = true;
		try {
			const cfg = await fetchConfig(selectedInstance || undefined);
			fullConfig = cfg.raw_yaml;
			const parsed = yaml.load(cfg.raw_yaml) as any;
			timeIntervals = parsed?.time_intervals ?? [];
		} catch (e) {
			toast.error('Erreur: ' + (e instanceof Error ? e.message : ''));
		} finally {
			loading = false;
		}
	}

	function addInterval() {
		timeIntervals = [...timeIntervals, {
			name: `interval-${timeIntervals.length + 1}`,
			time_intervals: [{ weekdays: [], times: [{ start_time: '22:00', end_time: '06:00' }] }]
		}];
	}

	function removeInterval(i: number) {
		timeIntervals = timeIntervals.filter((_, idx) => idx !== i);
	}

	function addTimeIntervalSpec(idx: number) {
		timeIntervals[idx].time_intervals = [
			...(timeIntervals[idx].time_intervals ?? []),
			{ weekdays: [], times: [{ start_time: '00:00', end_time: '23:59' }] }
		];
	}

	function removeTimeIntervalSpec(intervalIdx: number, specIdx: number) {
		timeIntervals[intervalIdx].time_intervals = timeIntervals[intervalIdx].time_intervals.filter((_, i) => i !== specIdx);
	}

	async function previewDiff() {
		try {
			const parsed = yaml.load(fullConfig) as any;
			parsed.time_intervals = timeIntervals.length > 0 ? timeIntervals : undefined;
			proposedYaml = yaml.dump(parsed, { lineWidth: 120 });
			diffResult = await diffConfig(selectedInstance || '', proposedYaml);
			step = 'diff';
		} catch (e) {
			toast.error('Erreur: ' + (e instanceof Error ? e.message : ''));
		}
	}

	async function save() {
		saving = true;
		try {
			await saveConfig({
				alertmanager: selectedInstance || '',
				raw_yaml: proposedYaml,
				save_mode: saveMode,
				disk_options: saveMode === 'disk' ? { file_path: diskPath } : undefined,
				git_options: saveMode !== 'disk' ? {
					repo: gitRepo, branch: gitBranch, file_path: gitFilePath
				} : undefined,
				webhook_url: webhookUrl || undefined
			});
			toast.success('Configuration sauvegardée');
			fullConfig = proposedYaml;
			step = 'edit';
			diffResult = null;
		} catch (e) {
			toast.error('Erreur de sauvegarde: ' + (e instanceof Error ? e.message : ''));
		} finally {
			saving = false;
		}
	}

	onMount(load);

	const WEEKDAYS = ['monday', 'tuesday', 'wednesday', 'thursday', 'friday', 'saturday', 'sunday'];
</script>

<div class="mt-4 space-y-4">
	<div class="flex items-center justify-between">
		<div>
			<h2 class="font-semibold">Time Intervals</h2>
			<p class="text-xs text-muted-foreground mt-0.5">Définissez des intervalles de temps nommés, utilisables dans les routes comme <code class="bg-muted px-1 rounded">mute_time_intervals</code> ou <code class="bg-muted px-1 rounded">active_time_intervals</code>.</p>
		</div>
		<select bind:value={selectedInstance} onchange={load} class="px-2 py-1 rounded border bg-background text-sm">
			<option value="">Défaut</option>
			{#each $instances as inst}
				<option value={inst.name}>{inst.name}</option>
			{/each}
		</select>
	</div>

	{#if loading}
		<div class="py-8 text-center text-muted-foreground animate-pulse">Chargement…</div>
	{:else if step === 'edit'}
		<div class="space-y-4">
			{#each timeIntervals as interval, i}
				<div class="p-4 rounded-lg border bg-card space-y-3">
					<div class="flex items-center justify-between">
						<input
							bind:value={interval.name}
							placeholder="Nom de l'intervalle"
							class="font-semibold px-2 py-1 rounded border bg-background text-sm flex-1 mr-4"
						/>
						<button onclick={() => removeInterval(i)} class="p-1 rounded text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors">
							<Trash2 class="h-4 w-4" />
						</button>
					</div>

					{#each interval.time_intervals as spec, j}
						<div class="pl-4 border-l-2 border-muted space-y-2">
							<div class="flex items-center justify-between">
								<span class="text-xs font-medium text-muted-foreground">Spécification {j + 1}</span>
								{#if interval.time_intervals.length > 1}
									<button onclick={() => removeTimeIntervalSpec(i, j)} class="p-0.5 rounded text-muted-foreground hover:text-destructive transition-colors">
										<Trash2 class="h-3 w-3" />
									</button>
								{/if}
							</div>

							<!-- Weekdays -->
							<div>
								<label class="text-xs text-muted-foreground block mb-1">Jours de la semaine</label>
								<div class="flex flex-wrap gap-2">
									{#each WEEKDAYS as day}
										<label class="flex items-center gap-1 text-xs cursor-pointer">
											<input
												type="checkbox"
												checked={spec.weekdays?.includes(day) ?? false}
												onchange={(e) => {
													const checked = (e.target as HTMLInputElement).checked;
													spec.weekdays = checked
														? [...(spec.weekdays ?? []), day]
														: (spec.weekdays ?? []).filter(d => d !== day);
												}}
											/>
											{day.slice(0, 3)}
										</label>
									{/each}
								</div>
							</div>

							<!-- Time ranges -->
							{#if spec.times}
								<div>
									<label class="text-xs text-muted-foreground block mb-1">Plages horaires</label>
									{#each spec.times as t}
										<div class="flex items-center gap-2">
											<label class="text-xs text-muted-foreground">De</label>
											<input type="time" bind:value={t.start_time} class="px-2 py-1 rounded border bg-background text-sm" />
											<label class="text-xs text-muted-foreground">à</label>
											<input type="time" bind:value={t.end_time} class="px-2 py-1 rounded border bg-background text-sm" />
										</div>
									{/each}
								</div>
							{/if}

							<!-- Location / timezone -->
							<div>
								<label class="text-xs text-muted-foreground block mb-1">Fuseau horaire (optionnel, ex: Europe/Paris)</label>
								<input
									bind:value={spec.location}
									placeholder="UTC"
									class="px-2 py-1 rounded border bg-background text-xs w-48"
								/>
							</div>
						</div>
					{/each}

					<button onclick={() => addTimeIntervalSpec(i)} class="flex items-center gap-1 text-xs text-primary hover:underline">
						<Plus class="h-3 w-3" /> Ajouter une spécification
					</button>
				</div>
			{/each}

			<button onclick={addInterval} class="flex items-center gap-2 px-4 py-2 rounded-md border text-sm hover:bg-muted transition-colors">
				<Plus class="h-4 w-4" />
				Ajouter un Time Interval
			</button>

			<button onclick={previewDiff} class="flex items-center gap-2 px-4 py-2 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-sm transition-colors">
				<Eye class="h-4 w-4" />
				Voir le diff
			</button>
		</div>

	{:else if step === 'diff' && diffResult}
		<YamlDiffViewer diff={diffResult.diff} hasChanges={diffResult.has_changes} />

		{#if diffResult.has_changes}
			<div class="p-4 rounded-lg border space-y-3 mt-3">
				<h3 class="font-semibold text-sm">Mode de sauvegarde</h3>
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
				<input bind:value={webhookUrl} placeholder="Webhook URL (optionnel)" class="w-full px-3 py-2 rounded border bg-background text-sm" />
			</div>
		{/if}

		<div class="flex gap-2 mt-3">
			<button onclick={() => step = 'edit'} class="px-4 py-2 rounded-md border text-sm hover:bg-muted">Retour</button>
			{#if diffResult.has_changes}
				<button onclick={save} disabled={saving} class="flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm disabled:opacity-50 transition-colors">
					<Save class="h-4 w-4" />
					{saving ? 'Sauvegarde...' : 'Confirmer et sauvegarder'}
				</button>
			{/if}
		</div>
	{/if}
</div>
