<script lang="ts">
	import { onMount } from 'svelte';
	import { diffConfig, saveConfig, fetchGitopsDefaults, fetchHistory } from '$lib/api/config';
	import type { GitopsDefaults, SaveRecord } from '$lib/api/types';
	import { configDraftStore } from '$lib/stores/configDraft';
	import { instances } from '$lib/stores/alerts';
	import { canEditConfig } from '$lib/stores/auth';
	import YamlDiffViewer from '$lib/components/config/YamlDiffViewer.svelte';
	import { toast } from 'svelte-sonner';
	import { Save, CheckCircle, AlertTriangle, Lock, ChevronDown, ChevronRight } from 'lucide-svelte';

	// ─── GitOps availability ──────────────────────────────────────────────────
	let gitopsDefaults = $state<GitopsDefaults | null>(null);

	// ─── Diff state ───────────────────────────────────────────────────────────
	let diffResult = $state<{ diff: string; has_changes: boolean } | null>(null);
	let diffLoading = $state(false);

	// ─── Save form state ──────────────────────────────────────────────────────
	let saving = $state(false);
	let saveMode = $state<'disk' | 'github' | 'gitlab'>('disk');
	let diskPath = $state('');
	let gitRepo = $state('');
	let gitBranch = $state('main');
	let gitFilePath = $state('alertmanager.yml');
	let gitCommitMessage = $state('');
	let gitAuthorName = $state('');
	let gitAuthorEmail = $state('');

	// ─── Result banners ───────────────────────────────────────────────────────
	let saveResult = $state<{
		mode: string;
		commit_sha?: string;
		html_url?: string;
		warning?: string;
	} | null>(null);
	let saveError = $state<string | null>(null);

	// ─── History ──────────────────────────────────────────────────────────────
	let history = $state<SaveRecord[]>([]);
	let expandedDiffs = $state<Record<number, { diff: string; has_changes: boolean } | null>>({});
	let expandLoading = $state<Record<number, boolean>>({});

	// ─── Derived ─────────────────────────────────────────────────────────────
	const draft = $derived($configDraftStore);

	const canSave = $derived(
		!!draft &&
		!!diffResult?.has_changes &&
		!saving &&
		$canEditConfig &&
		(saveMode === 'disk'
			? !!diskPath
			: !!gitRepo && !!gitBranch && !!gitFilePath)
	);

	// Auto-refresh diff whenever draft changes.
	$effect(() => {
		const d = $configDraftStore;
		if (d) {
			loadDiff(d.instance, d.rawYaml);
		} else {
			diffResult = null;
		}
	});

	// ─── Functions ────────────────────────────────────────────────────────────

	async function loadDiff(instance: string, rawYaml: string) {
		diffLoading = true;
		try {
			diffResult = await diffConfig(instance, rawYaml);
		} catch (e) {
			toast.error('Diff error: ' + (e instanceof Error ? e.message : ''));
		} finally {
			diffLoading = false;
		}
	}

	async function loadHistory(instance: string) {
		try {
			const resp = await fetchHistory(instance);
			history = resp.history;
		} catch {
			// Non-fatal — history section stays empty.
		}
	}

	async function save() {
		if (!draft) return;
		saving = true;
		saveResult = null;
		saveError = null;
		try {
			const result = await saveConfig({
				alertmanager: draft.instance,
				raw_yaml: draft.rawYaml,
				save_mode: saveMode,
				disk_options: saveMode === 'disk' ? { file_path: diskPath } : undefined,
				git_options: saveMode !== 'disk'
					? {
						repo: gitRepo,
						branch: gitBranch,
						file_path: gitFilePath,
						commit_message: gitCommitMessage || undefined,
						author_name: gitAuthorName || undefined,
						author_email: gitAuthorEmail || undefined
					}
					: undefined
			});
			saveResult = {
				mode: result.mode,
				commit_sha: result.commit_sha,
				html_url: result.html_url,
				warning: result.warning
			};
			toast.success('Configuration saved successfully');
			// Refresh diff and history after a successful save.
			await Promise.all([
				loadDiff(draft.instance, draft.rawYaml),
				loadHistory(draft.instance)
			]);
		} catch (e) {
			saveError = e instanceof Error ? e.message : 'Save failed';
		} finally {
			saving = false;
		}
	}

	function isModeDisabled(mode: 'disk' | 'github' | 'gitlab'): boolean {
		if (!gitopsDefaults) return false; // unknown — allow, server will validate
		if (mode === 'github') return !gitopsDefaults.github_configured;
		if (mode === 'gitlab') return !gitopsDefaults.gitlab_configured;
		return false; // disk is always available
	}

	function modeTooltip(mode: 'disk' | 'github' | 'gitlab'): string {
		if (mode === 'github' && gitopsDefaults && !gitopsDefaults.github_configured) {
			return 'GitHub token not configured in alertlens.yaml';
		}
		if (mode === 'gitlab' && gitopsDefaults && !gitopsDefaults.gitlab_configured) {
			return 'GitLab token not configured in alertlens.yaml';
		}
		return '';
	}

	async function toggleExpandDiff(idx: number, record: SaveRecord) {
		if (expandedDiffs[idx] !== undefined) {
			// Collapse
			const next = { ...expandedDiffs };
			delete next[idx];
			expandedDiffs = next;
			return;
		}
		expandLoading = { ...expandLoading, [idx]: true };
		try {
			const result = await diffConfig(record.alertmanager, record.raw_yaml);
			expandedDiffs = { ...expandedDiffs, [idx]: result };
		} catch {
			expandedDiffs = { ...expandedDiffs, [idx]: null };
		} finally {
			const next = { ...expandLoading };
			delete next[idx];
			expandLoading = next;
		}
	}

	function formatTime(isoString: string): string {
		try {
			return new Date(isoString).toLocaleString();
		} catch {
			return isoString;
		}
	}

	const modeBadgeClass: Record<string, string> = {
		disk:   'bg-gray-100 text-gray-700 dark:bg-gray-800 dark:text-gray-300',
		github: 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300',
		gitlab: 'bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-300'
	};

	onMount(async () => {
		// Load GitOps defaults and initial history in parallel.
		const currentInstance = $configDraftStore?.instance ?? $instances[0]?.name ?? '';
		await Promise.all([
			fetchGitopsDefaults(currentInstance).then(d => {
				gitopsDefaults = d;
				// Pre-fill the disk path from the instance's config_file_path if not already set.
				if (!diskPath && d.disk_file_path) {
					diskPath = d.disk_file_path;
				}
			}).catch(() => {}),
			loadHistory(currentInstance)
		]);
	});
</script>

<div class="mt-4 space-y-6">

	{#if !$canEditConfig}
		<div class="flex items-center gap-2 p-3 rounded-lg border bg-muted/30 text-sm text-muted-foreground">
			<Lock class="h-4 w-4 shrink-0" />
			You need the <strong class="mx-1 text-foreground">config-editor</strong> role to save configurations.
		</div>
	{/if}

	<!-- ─── Diff preview ─────────────────────────────────────────────────── -->
	<div class="space-y-2">
		<h2 class="font-semibold">Pending changes</h2>

		{#if !draft}
			<div class="p-4 rounded-lg border bg-muted/30 text-sm text-muted-foreground text-center">
				No pending changes. Edit the routing, receivers, or time intervals first.
			</div>
		{:else if diffLoading}
			<div class="py-6 text-center text-muted-foreground animate-pulse text-sm">Computing diff…</div>
		{:else if diffResult}
			<YamlDiffViewer diff={diffResult.diff} hasChanges={diffResult.has_changes} />
		{/if}
	</div>

	<!-- ─── Save form ───────────────────────────────────────────────────── -->
	{#if $canEditConfig && draft}
		<div class="rounded-lg border bg-card p-4 space-y-4">
			<h2 class="font-semibold">Save mode</h2>

			<!-- Mode selector -->
			<div class="flex gap-4">
				{#each (['disk', 'github', 'gitlab'] as const) as mode}
					{@const disabled = isModeDisabled(mode)}
					{@const tip = modeTooltip(mode)}
					<label
						class="flex items-center gap-1.5 {disabled ? 'opacity-40 cursor-not-allowed' : 'cursor-pointer'}"
						title={tip}
					>
						<input
							type="radio"
							bind:group={saveMode}
							value={mode}
							disabled={disabled}
						/>
						<span class="text-sm capitalize">{mode}</span>
					</label>
				{/each}
			</div>

			<!-- Conditional fields -->
			{#if saveMode === 'disk'}
				<div class="space-y-1">
					<label class="text-xs font-medium text-muted-foreground">Config file path</label>
					<input
						bind:value={diskPath}
						placeholder="/etc/alertmanager/alertmanager.yml"
						class="w-full px-3 py-2 rounded border bg-background text-sm"
					/>
				</div>
			{:else}
				<div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
					<div class="space-y-1 sm:col-span-2">
						<label class="text-xs font-medium text-muted-foreground">Repository</label>
						<input
							bind:value={gitRepo}
							placeholder={saveMode === 'github' ? 'owner/repo' : 'namespace/project'}
							class="w-full px-3 py-2 rounded border bg-background text-sm"
						/>
					</div>
					<div class="space-y-1">
						<label class="text-xs font-medium text-muted-foreground">Branch</label>
						<input bind:value={gitBranch} placeholder="main" class="w-full px-3 py-2 rounded border bg-background text-sm" />
					</div>
					<div class="space-y-1">
						<label class="text-xs font-medium text-muted-foreground">File path</label>
						<input bind:value={gitFilePath} placeholder="alertmanager.yml" class="w-full px-3 py-2 rounded border bg-background text-sm" />
					</div>
					<div class="space-y-1 sm:col-span-2">
						<label class="text-xs font-medium text-muted-foreground">Commit message</label>
						<input bind:value={gitCommitMessage} placeholder="chore: update alertmanager config" class="w-full px-3 py-2 rounded border bg-background text-sm" />
					</div>
					<div class="space-y-1">
						<label class="text-xs font-medium text-muted-foreground">Author name</label>
						<input bind:value={gitAuthorName} placeholder="AlertLens" class="w-full px-3 py-2 rounded border bg-background text-sm" />
					</div>
					<div class="space-y-1">
						<label class="text-xs font-medium text-muted-foreground">Author email</label>
						<input bind:value={gitAuthorEmail} placeholder="alertlens@example.com" class="w-full px-3 py-2 rounded border bg-background text-sm" />
					</div>
				</div>
			{/if}

			<!-- Save button -->
			<div class="flex items-center gap-3 pt-1">
				<button
					onclick={save}
					disabled={!canSave}
					class="flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm disabled:opacity-40 disabled:cursor-not-allowed transition-colors"
				>
					<Save class="h-4 w-4" />
					{saving ? 'Saving…' : 'Save & Deploy'}
				</button>
				{#if diffResult && !diffResult.has_changes}
					<span class="text-sm text-muted-foreground">No changes to save.</span>
				{/if}
			</div>

			<!-- Success banner -->
			{#if saveResult}
				<div class="flex items-start gap-2 p-3 rounded-md bg-green-50 dark:bg-green-900/20 border border-green-200 dark:border-green-800 text-sm">
					<CheckCircle class="h-4 w-4 text-green-600 dark:text-green-400 shrink-0 mt-0.5" />
					<div>
						<p class="font-medium text-green-800 dark:text-green-300">
							Saved via <span class="capitalize">{saveResult.mode}</span>
							{#if saveResult.commit_sha}
								— commit <code class="font-mono text-xs">{saveResult.commit_sha.slice(0, 8)}</code>
							{/if}
						</p>
						{#if saveResult.html_url}
							<a
								href={saveResult.html_url}
								target="_blank"
								rel="noopener noreferrer"
								class="text-xs text-green-700 dark:text-green-400 underline hover:no-underline"
							>
								View commit ↗
							</a>
						{/if}
						{#if saveResult.warning}
							<p class="text-xs text-yellow-700 dark:text-yellow-400 mt-1">⚠ {saveResult.warning}</p>
						{/if}
					</div>
				</div>
			{/if}

			<!-- Error banner -->
			{#if saveError}
				<div class="flex items-start gap-2 p-3 rounded-md bg-destructive/10 border border-destructive/30 text-sm">
					<AlertTriangle class="h-4 w-4 text-destructive shrink-0 mt-0.5" />
					<p class="text-destructive">{saveError}</p>
				</div>
			{/if}
		</div>
	{/if}

	<!-- ─── Save History ─────────────────────────────────────────────────── -->
	<div class="space-y-2">
		<h2 class="font-semibold">Save History</h2>
		<p class="text-xs text-muted-foreground">In-memory since last restart · up to 50 saves per instance · newest first</p>

		{#if history.length === 0}
			<div class="p-4 rounded-lg border bg-muted/30 text-sm text-muted-foreground text-center">
				No saves recorded since last restart.
			</div>
		{:else}
			<div class="space-y-2">
				{#each history as record, idx}
					<div class="rounded-lg border bg-card overflow-hidden">
						<!-- Row header -->
						<div class="flex items-center gap-3 px-4 py-3">
							<span class="text-xs text-muted-foreground shrink-0">{formatTime(record.saved_at)}</span>
							<span class="px-2 py-0.5 rounded-full text-xs font-medium {modeBadgeClass[record.mode] ?? modeBadgeClass.disk}">
								{record.mode}
							</span>
							<span class="text-xs text-muted-foreground">{record.actor}</span>
							{#if record.html_url}
								<a
									href={record.html_url}
									target="_blank"
									rel="noopener noreferrer"
									class="text-xs text-primary underline hover:no-underline ml-auto shrink-0"
								>
									commit ↗
								</a>
							{/if}
							<button
								onclick={() => toggleExpandDiff(idx, record)}
								disabled={expandLoading[idx]}
								class="flex items-center gap-1 ml-auto shrink-0 text-xs text-muted-foreground hover:text-foreground transition-colors disabled:opacity-50"
							>
								{#if expandLoading[idx]}
									<span class="animate-pulse">Loading…</span>
								{:else if expandedDiffs[idx] !== undefined}
									<ChevronDown class="h-3.5 w-3.5" />
									Collapse diff
								{:else}
									<ChevronRight class="h-3.5 w-3.5" />
									Expand diff
								{/if}
							</button>
						</div>

						<!-- Inline diff (lazy-loaded on first expand) -->
						{#if expandedDiffs[idx] !== undefined}
							<div class="border-t px-4 py-3">
								{#if expandedDiffs[idx] === null}
									<p class="text-xs text-destructive">Failed to load diff.</p>
								{:else}
									<YamlDiffViewer
										diff={expandedDiffs[idx]!.diff}
										hasChanges={expandedDiffs[idx]!.has_changes}
									/>
								{/if}
							</div>
						{/if}
					</div>
				{/each}
			</div>
		{/if}
	</div>

</div>
