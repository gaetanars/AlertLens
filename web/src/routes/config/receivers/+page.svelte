<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchConfig, diffConfig, validateConfig, saveConfig } from '$lib/api/config';
	import YamlDiffViewer from '$lib/components/config/YamlDiffViewer.svelte';
	import { instances } from '$lib/stores/alerts';
	import { toast } from 'svelte-sonner';
	import { Plus, Trash2, Eye, Save } from 'lucide-svelte';
	import * as yaml from 'js-yaml';

	let selectedInstance = $state($instances[0]?.name ?? '');
	let fullConfig = $state('');
	let receivers = $state<Receiver[]>([]);
	let loading = $state(false);
	let saving = $state(false);
	let diffResult = $state<{ diff: string; has_changes: boolean } | null>(null);
	let step = $state<'edit' | 'diff'>('edit');
	let editingIdx = $state<number | null>(null);
	let proposedYaml = $state('');

	// Save options (SPEC-05)
	let saveMode = $state<'disk' | 'github' | 'gitlab'>('disk');
	let diskPath = $state('');
	let gitRepo = $state('');
	let gitBranch = $state('main');
	let gitFilePath = $state('alertmanager.yml');
	let webhookUrl = $state('');

	interface WebhookConfig { url: string; send_resolved?: boolean; }
	interface SlackConfig { api_url: string; channel: string; title?: string; }
	interface PagerdutyConfig { routing_key: string; severity?: string; }
	interface EmailConfig { to: string; from?: string; smarthost?: string; }
	type ReceiverConfig =
		| { type: 'webhook'; config: WebhookConfig }
		| { type: 'slack'; config: SlackConfig }
		| { type: 'pagerduty'; config: PagerdutyConfig }
		| { type: 'email'; config: EmailConfig };

	interface Receiver {
		name: string;
		configs: ReceiverConfig[];
	}

	function parseReceivers(rawYaml: string): Receiver[] {
		try {
			const parsed = yaml.load(rawYaml) as any;
			if (!parsed?.receivers) return [];
			return (parsed.receivers as any[]).map((r: any) => ({
				name: r.name ?? '',
				configs: [
					...(r.webhook_configs ?? []).map((c: any) => ({ type: 'webhook' as const, config: c })),
					...(r.slack_configs ?? []).map((c: any) => ({ type: 'slack' as const, config: c })),
					...(r.pagerduty_configs ?? []).map((c: any) => ({ type: 'pagerduty' as const, config: c })),
					...(r.email_configs ?? []).map((c: any) => ({ type: 'email' as const, config: c })),
				]
			}));
		} catch { return []; }
	}

	function buildYaml(): string {
		try {
			const parsed = yaml.load(fullConfig) as any ?? {};
			parsed.receivers = receivers.map(r => {
				const rec: any = { name: r.name };
				r.configs.forEach(c => {
					const key = c.type + '_configs';
					if (!rec[key]) rec[key] = [];
					rec[key].push(c.config);
				});
				return rec;
			});
			return yaml.dump(parsed, { lineWidth: 120 });
		} catch { return fullConfig; }
	}

	async function load() {
		loading = true;
		try {
			const cfg = await fetchConfig(selectedInstance || undefined);
			fullConfig = cfg.raw_yaml;
			receivers = parseReceivers(cfg.raw_yaml);
		} catch (e) {
			toast.error('Error: ' + (e instanceof Error ? e.message : ''));
		} finally { loading = false; }
	}

	function addReceiver() {
		receivers = [...receivers, { name: `receiver-${receivers.length + 1}`, configs: [] }];
		editingIdx = receivers.length - 1;
	}

	function removeReceiver(i: number) {
		receivers = receivers.filter((_, idx) => idx !== i);
		editingIdx = null;
	}

	function addConfig(i: number, type: ReceiverConfig['type']) {
		const defaults: Record<string, any> = {
			webhook: { url: '', send_resolved: true },
			slack: { api_url: '', channel: '#alerts' },
			pagerduty: { routing_key: '', severity: 'critical' },
			email: { to: '' }
		};
		receivers[i].configs = [...receivers[i].configs, { type, config: defaults[type] } as ReceiverConfig];
	}

	async function previewDiff() {
		proposedYaml = buildYaml();
		try {
			diffResult = await diffConfig(selectedInstance || '', proposedYaml);
			step = 'diff';
		} catch (e) { toast.error('Error: ' + (e instanceof Error ? e.message : '')); }
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
			toast.success('Configuration saved');
			fullConfig = proposedYaml;
			step = 'edit';
			diffResult = null;
		} catch (e) {
			toast.error('Save error: ' + (e instanceof Error ? e.message : ''));
		} finally {
			saving = false;
		}
	}

	onMount(load);
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

	{#if loading}
		<div class="py-8 text-center text-muted-foreground animate-pulse">Loading…</div>
	{:else if step === 'edit'}
		<div class="grid grid-cols-1 lg:grid-cols-3 gap-4">
			<!-- Receiver list -->
			<div class="space-y-2">
				{#each receivers as r, i}
					<div
						class="flex items-center justify-between px-3 py-2 rounded-lg border cursor-pointer transition-colors
							{editingIdx === i ? 'bg-primary/5 border-primary' : 'hover:bg-muted'}"
						onclick={() => editingIdx = i}
						role="button"
						tabindex="0"
					>
						<span class="text-sm font-medium">{r.name}</span>
						<div class="flex items-center gap-2">
							<span class="text-xs text-muted-foreground">{r.configs.length} config{r.configs.length !== 1 ? 's' : ''}</span>
							<button onclick={(e) => { e.stopPropagation(); removeReceiver(i); }} class="p-1 rounded text-muted-foreground hover:text-destructive transition-colors">
								<Trash2 class="h-3.5 w-3.5" />
							</button>
						</div>
					</div>
				{/each}
				<button onclick={addReceiver} class="w-full flex items-center justify-center gap-2 px-3 py-2 rounded-lg border border-dashed text-sm text-muted-foreground hover:text-foreground hover:border-foreground transition-colors">
					<Plus class="h-4 w-4" />
					Add receiver
				</button>
			</div>

			<!-- Receiver editor -->
			{#if editingIdx !== null && receivers[editingIdx]}
				{@const r = receivers[editingIdx]}
				<div class="lg:col-span-2 space-y-3">
					<div>
						<label class="text-sm font-medium mb-1 block">Name</label>
						<input bind:value={r.name} class="w-full px-3 py-2 rounded-md border bg-background text-sm" />
					</div>

					{#each r.configs as cfg, ci}
						<div class="p-3 rounded-lg border space-y-2">
							<div class="flex items-center justify-between">
								<span class="text-sm font-medium capitalize">{cfg.type}</span>
								<button onclick={() => { r.configs = r.configs.filter((_, idx) => idx !== ci); }} class="p-1 rounded text-muted-foreground hover:text-destructive">
									<Trash2 class="h-3.5 w-3.5" />
								</button>
							</div>
							{#if cfg.type === 'webhook'}
								<div>
									<label class="text-xs text-muted-foreground">URL <span class="text-destructive">*</span></label>
									<input bind:value={cfg.config.url} placeholder="https://..." class="w-full px-2 py-1.5 rounded border bg-background text-sm mt-0.5" />
								</div>
							{:else if cfg.type === 'slack'}
								<div class="grid grid-cols-2 gap-2">
									<div>
										<label class="text-xs text-muted-foreground">API URL <span class="text-destructive">*</span></label>
										<input bind:value={cfg.config.api_url} placeholder="https://hooks.slack.com/..." class="w-full px-2 py-1.5 rounded border bg-background text-sm mt-0.5" />
									</div>
									<div>
										<label class="text-xs text-muted-foreground">Channel <span class="text-destructive">*</span></label>
										<input bind:value={cfg.config.channel} placeholder="#alerts" class="w-full px-2 py-1.5 rounded border bg-background text-sm mt-0.5" />
									</div>
								</div>
							{:else if cfg.type === 'pagerduty'}
								<div>
									<label class="text-xs text-muted-foreground">Routing Key <span class="text-destructive">*</span></label>
									<input bind:value={cfg.config.routing_key} placeholder="xxxxxxxxxxxxxxxx" class="w-full px-2 py-1.5 rounded border bg-background text-sm mt-0.5" />
								</div>
							{:else if cfg.type === 'email'}
								<div>
									<label class="text-xs text-muted-foreground">Recipient <span class="text-destructive">*</span></label>
									<input bind:value={cfg.config.to} placeholder="team@example.com" class="w-full px-2 py-1.5 rounded border bg-background text-sm mt-0.5" />
								</div>
							{/if}
						</div>
					{/each}

					<div class="flex flex-wrap gap-2">
						{#each ['webhook', 'slack', 'pagerduty', 'email'] as type}
							<button
								onclick={() => addConfig(editingIdx!, type as any)}
								class="flex items-center gap-1 px-3 py-1 rounded-full border text-xs hover:bg-muted transition-colors"
							>
								<Plus class="h-3 w-3" />
								{type}
							</button>
						{/each}
					</div>
				</div>
			{/if}
		</div>

		<button onclick={previewDiff} class="flex items-center gap-2 px-4 py-2 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-sm transition-colors">
			<Eye class="h-4 w-4" />
			Preview diff
		</button>

	{:else if step === 'diff' && diffResult}
		<YamlDiffViewer diff={diffResult.diff} hasChanges={diffResult.has_changes} />

		{#if diffResult.has_changes}
			<!-- SPEC-05: save options in the diff step -->
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
			<button onclick={() => step = 'edit'} class="px-4 py-2 rounded-md border text-sm hover:bg-muted">Back</button>
			{#if diffResult.has_changes}
				<button onclick={save} disabled={saving} class="flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm disabled:opacity-50 transition-colors">
					<Save class="h-4 w-4" />
					{saving ? 'Saving...' : 'Confirm and save'}
				</button>
			{/if}
		</div>
	{/if}
</div>
