<script lang="ts">
	import { onMount } from 'svelte';
	import { diffConfig, saveConfig } from '$lib/api/config';
	import {
		listReceivers,
		upsertReceiver,
		deleteReceiver,
		validateReceiver,
		getReceiverRoutes,
		exportConfig
	} from '$lib/api/builder';
	import type { ReceiverDef, BuilderReceiverRouteRef } from '$lib/api/types';
	import ReceiverEditor from '$lib/components/config/ReceiverEditor.svelte';
	import YamlDiffViewer from '$lib/components/config/YamlDiffViewer.svelte';
	import { instances } from '$lib/stores/alerts';
	import { canEditConfig } from '$lib/stores/auth';
	import { toast } from 'svelte-sonner';
	import { Plus, Trash2, Eye, Save, Lock } from 'lucide-svelte';

	let selectedInstance = $state($instances[0]?.name ?? '');
	let receivers = $state<ReceiverDef[]>([]);
	let editing = $state<ReceiverDef | null>(null);
	let loading = $state(false);
	let saving = $state(false);
	let upserting = $state(false);
	let validationErrors = $state<string[]>([]);

	// Delete guard state
	let deleteGuard = $state<{
		name: string;
		refs: BuilderReceiverRouteRef[];
	} | null>(null);

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

	// Debounced inline validation
	let validationTimer: ReturnType<typeof setTimeout> | null = null;
	$effect(() => {
		const snap = editing;
		if (!snap || !$canEditConfig) return;
		if (validationTimer) clearTimeout(validationTimer);
		validationTimer = setTimeout(async () => {
			try {
				const result = await validateReceiver(snap);
				validationErrors = result.valid ? [] : (result.errors ?? []);
			} catch {
				// ignore transient errors
			}
		}, 500);
	});

	async function load() {
		loading = true;
		try {
			const resp = await listReceivers(selectedInstance || undefined);
			receivers = resp.receivers;
		} catch (e) {
			toast.error('Load error: ' + (e instanceof Error ? e.message : ''));
		} finally {
			loading = false;
		}
	}

	function selectReceiver(r: ReceiverDef) {
		editing = structuredClone(r);
		deleteGuard = null;
		validationErrors = [];
		step = 'edit';
	}

	function addReceiver() {
		editing = {
			name: '',
			webhook_configs: [],
			slack_configs: [],
			email_configs: [],
			pagerduty_configs: [],
			opsgenie_configs: []
		};
		deleteGuard = null;
		validationErrors = [];
		step = 'edit';
	}

	async function saveReceiver() {
		if (!editing) return;
		upserting = true;
		try {
			await upsertReceiver(editing.name, editing, selectedInstance || undefined);
			toast.success(`Receiver "${editing.name}" saved`);
			await load();
		} catch (e) {
			toast.error('Save error: ' + (e instanceof Error ? e.message : ''));
		} finally {
			upserting = false;
		}
	}

	async function initiateDelete(name: string) {
		deleteGuard = null;
		try {
			const resp = await getReceiverRoutes(name, selectedInstance || undefined);
			if (resp.referenced_by.length > 0) {
				deleteGuard = { name, refs: resp.referenced_by };
			} else {
				await confirmDelete(name);
			}
		} catch (e) {
			toast.error('Error: ' + (e instanceof Error ? e.message : ''));
		}
	}

	async function confirmDelete(name: string) {
		try {
			await deleteReceiver(name, selectedInstance || undefined);
			toast.success(`Receiver "${name}" deleted`);
			if (editing?.name === name) editing = null;
			deleteGuard = null;
			await load();
		} catch (e) {
			toast.error('Delete error: ' + (e instanceof Error ? e.message : ''));
		}
	}

	async function previewDiff() {
		if (!editing) return;
		try {
			const exported = await exportConfig({
				instance: selectedInstance || undefined,
				receivers: [editing]
			});
			pendingYaml = exported.raw_yaml;
			diffResult = await diffConfig(selectedInstance || '', exported.raw_yaml);
			step = 'diff';
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

	onMount(() => {
		load();
	});
</script>

<div class="mt-4 space-y-4">
	<div class="flex items-center justify-between">
		<h2 class="font-semibold">Receivers</h2>
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
			You need the <strong class="mx-1 text-foreground">config-editor</strong> role to edit receivers.
		</div>
	{/if}

	{#if loading}
		<div class="py-8 text-center text-muted-foreground animate-pulse">Loading…</div>
	{:else if step === 'edit'}
		<div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
			<!-- Receiver list -->
			<div class="space-y-2">
				{#each receivers as r}
					<div
						class="flex items-center justify-between px-3 py-2 rounded-lg border cursor-pointer transition-colors
							{editing?.name === r.name ? 'bg-primary/5 border-primary' : 'hover:bg-muted'}"
						onclick={() => selectReceiver(r)}
						role="button"
						tabindex="0"
						onkeydown={(e) => e.key === 'Enter' && selectReceiver(r)}
					>
						<span class="text-sm font-medium truncate">{r.name}</span>
						{#if $canEditConfig}
							<button
								onclick={(e) => { e.stopPropagation(); initiateDelete(r.name); }}
								class="p-1 rounded text-muted-foreground hover:text-destructive transition-colors shrink-0"
								title="Delete receiver"
							>
								<Trash2 class="h-3.5 w-3.5" />
							</button>
						{/if}
					</div>
				{/each}

				{#if $canEditConfig}
					<button
						onclick={addReceiver}
						class="w-full flex items-center justify-center gap-2 px-3 py-2 rounded-lg border border-dashed text-sm text-muted-foreground hover:text-foreground hover:border-foreground transition-colors"
					>
						<Plus class="h-4 w-4" />
						Add receiver
					</button>
				{/if}
			</div>

			<!-- Receiver editor -->
			{#if editing !== null}
				<div class="lg:col-span-2 space-y-3">
					<!-- Name field -->
					<div>
						<label class="text-sm font-medium mb-1 block">Name</label>
						{#if $canEditConfig}
							<input
								value={editing.name}
								oninput={(e) => { if (editing) editing = { ...editing, name: (e.target as HTMLInputElement).value }; }}
								placeholder="my-receiver"
								class="w-full px-3 py-2 rounded-md border bg-background text-sm"
							/>
						{:else}
							<p class="px-3 py-2 rounded-md border bg-muted/30 text-sm font-mono">{editing.name}</p>
						{/if}
					</div>

					<!-- Integration editor -->
					<ReceiverEditor
						receiver={editing}
						onUpdate={(r) => { editing = r; }}
						{validationErrors}
						readonly={!$canEditConfig}
					/>

					<!-- Delete guard: inline confirmation -->
					{#if deleteGuard}
						<div class="rounded-md border border-destructive/50 bg-destructive/10 p-3 space-y-2">
							<p class="text-sm font-medium text-destructive">
								"{deleteGuard.name}" is referenced by {deleteGuard.refs.length} route{deleteGuard.refs.length !== 1 ? 's' : ''}:
							</p>
							<ul class="space-y-1">
								{#each deleteGuard.refs as ref}
									<li class="text-xs text-muted-foreground">
										depth {ref.depth} — {ref.matchers.length ? ref.matchers.join(', ') : '(root route)'}
									</li>
								{/each}
							</ul>
							<div class="flex gap-2 pt-1">
								<button
									onclick={() => confirmDelete(deleteGuard!.name)}
									class="px-3 py-1.5 rounded-md bg-destructive text-destructive-foreground text-xs hover:bg-destructive/90 transition-colors"
								>
									Delete anyway
								</button>
								<button
									onclick={() => deleteGuard = null}
									class="px-3 py-1.5 rounded-md border text-xs hover:bg-muted transition-colors"
								>
									Cancel
								</button>
							</div>
						</div>
					{/if}

					{#if $canEditConfig}
						<div class="flex gap-2 flex-wrap">
							<button
								onclick={saveReceiver}
								disabled={upserting}
								class="flex items-center gap-2 px-4 py-2 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-sm disabled:opacity-50 transition-colors"
							>
								<Save class="h-4 w-4" />
								{upserting ? 'Saving…' : 'Save receiver'}
							</button>
							<button
								onclick={previewDiff}
								class="flex items-center gap-2 px-4 py-2 rounded-md border text-sm hover:bg-muted transition-colors"
							>
								<Eye class="h-4 w-4" />
								Preview diff
							</button>
						</div>
					{/if}
				</div>
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
