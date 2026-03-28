<script lang="ts">
	import { onMount } from 'svelte';
	import { fetchConfig, validateConfig, diffConfig, saveConfig } from '$lib/api/config';
	import { fetchRouting } from '$lib/api/routing';
	import { getRoute, listReceivers } from '$lib/api/builder';
	import type { RouteSpec } from '$lib/api/types';
	import RoutingTree from '$lib/components/routing/RoutingTree.svelte';
	import YamlDiffViewer from '$lib/components/config/YamlDiffViewer.svelte';
	import RouteNodeEditor, { type RouteFormNode, emptyNode } from '$lib/components/config/RouteNodeEditor.svelte';
	import { instances } from '$lib/stores/alerts';
	import { canEditConfig } from '$lib/stores/auth';
	import { configDraftStore } from '$lib/stores/configDraft';
	import { toast } from 'svelte-sonner';
	import type { RouteNode } from '$lib/api/types';
	import { Save, Eye, AlertTriangle, FormInput, Code, Lock } from 'lucide-svelte';
	import * as yaml from 'js-yaml';

	let selectedInstance = $state($instances[0]?.name ?? '');
	let rawYaml = $state('');
	let originalYaml = $state('');
	let loading = $state(false);
	let saving = $state(false);
	let step = $state<'edit' | 'diff'>('edit');
	let diffResult = $state<{ diff: string; has_changes: boolean } | null>(null);
	let validationErrors = $state<string[]>([]);
	let routeData = $state<{ alertmanager: string; route: RouteNode } | null>(null);
	let availableTimeIntervals = $state<string[]>([]);
	let availableReceivers = $state<string[]>([]);

	// SPEC-01: toggle between YAML and visual form editor.
	let editorTab = $state<'yaml' | 'form'>('yaml');
	let formRoute = $state<RouteFormNode>(emptyNode());

	// Live YAML derived from form state — updates on every form change without tab switching.
	// Uses rawYaml as the base so non-route config sections (global, receivers, …) are preserved.
	const formYaml = $derived.by(() => {
		try {
			const base = (yaml.load(rawYaml) as any) ?? {};
			base.route = formRouteToYaml(formRoute);
			return yaml.dump(base, { lineWidth: 120 });
		} catch {
			return rawYaml;
		}
	});

	// Stores the YAML that was sent to previewDiff so save() uses the exact same content.
	let pendingYaml = $state('');

	// Save options
	let saveMode = $state<'disk' | 'github' | 'gitlab'>('disk');
	let diskPath = $state('');
	let gitRepo = $state('');
	let gitBranch = $state('main');
	let gitFilePath = $state('alertmanager.yml');
	let webhookUrl = $state('');

	// Convert a RouteSpec (builder API, string matchers) to a RouteFormNode.
	function routeSpecToForm(r: RouteSpec): RouteFormNode {
		return {
			receiver: r.receiver ?? '',
			continue: r.continue ?? false,
			group_by: r.group_by ?? [],
			group_wait: r.group_wait ?? '',
			group_interval: r.group_interval ?? '',
			repeat_interval: r.repeat_interval ?? '',
			matchers: (r.matchers ?? []).map((m) => {
				const match = m.match(/^(\w+)(=~|!=|!~|=)"?(.*?)"?$/);
				if (match) {
					const [, name, op, value] = match;
					return { name, value, isRegex: op === '=~' || op === '!~', isEqual: op === '=' || op === '=~' };
				}
				return { name: m, value: '', isRegex: false, isEqual: true };
			}),
			mute_time_intervals: r.mute_time_intervals ?? [],
			active_time_intervals: r.active_time_intervals ?? [],
			routes: (r.routes ?? []).map(routeSpecToForm)
		};
	}

	// Collect all unique time interval names referenced anywhere in the route tree.
	function collectTimeIntervals(r: RouteSpec): string[] {
		const names = new Set<string>();
		function walk(node: RouteSpec) {
			(node.mute_time_intervals ?? []).forEach(n => names.add(n));
			(node.active_time_intervals ?? []).forEach(n => names.add(n));
			(node.routes ?? []).forEach(walk);
		}
		walk(r);
		return [...names];
	}

	async function load() {
		loading = true;
		try {
			const instance = selectedInstance || undefined;
			const [routeResp, receiversResp, cfg, routing] = await Promise.all([
				getRoute(instance),
				listReceivers(instance),
				fetchConfig(instance),
				fetchRouting(instance).catch(() => null)
			]);
			// Form tab: structured data from the builder API.
			formRoute = routeSpecToForm(routeResp.route);
			availableReceivers = receiversResp.receivers.map(r => r.name);
			availableTimeIntervals = collectTimeIntervals(routeResp.route);
			// YAML tab: raw config text.
			rawYaml = cfg.raw_yaml;
			originalYaml = cfg.raw_yaml;
			// Right panel: read-only routing tree preview.
			routeData = routing;
		} catch (e) {
			toast.error('Load error: ' + (e instanceof Error ? e.message : ''));
		} finally {
			loading = false;
		}
	}

	async function validate(yamlContent: string) {
		try {
			const r = await validateConfig(yamlContent);
			if (!r.valid) { validationErrors = r.errors ?? []; return false; }
			validationErrors = [];
			return true;
		} catch { return false; }
	}

	async function previewDiff() {
		// In form tab use the live derived YAML; in YAML tab use the editor content.
		const yamlToUse = editorTab === 'form' ? formYaml : rawYaml;
		if (!(await validate(yamlToUse))) return;
		try {
			pendingYaml = yamlToUse;
			diffResult = await diffConfig(selectedInstance || '', yamlToUse);
			step = 'diff';
			// Share the assembled YAML with the Save & Deploy tab.
			configDraftStore.set({ instance: selectedInstance, rawYaml: yamlToUse });
		} catch (e) {
			toast.error('Diff error: ' + (e instanceof Error ? e.message : ''));
		}
	}

	async function save() {
		saving = true;
		try {
			await saveConfig({
				alertmanager: selectedInstance || '',
				raw_yaml: pendingYaml || rawYaml,
				save_mode: saveMode,
				disk_options: saveMode === 'disk' ? { file_path: diskPath } : undefined,
				git_options: saveMode !== 'disk' ? {
					repo: gitRepo, branch: gitBranch, file_path: gitFilePath
				} : undefined,
				webhook_url: webhookUrl || undefined
			});
			toast.success('Configuration saved successfully');
			// Refresh form state from server so it reflects the persisted config.
			const routeResp = await getRoute(selectedInstance || undefined);
			formRoute = routeSpecToForm(routeResp.route);
			rawYaml = pendingYaml || rawYaml;
			originalYaml = rawYaml;
			step = 'edit';
			diffResult = null;
			pendingYaml = '';
		} catch (e) {
			toast.error('Save error: ' + (e instanceof Error ? e.message : ''));
		} finally {
			saving = false;
		}
	}

	// SPEC-01: convert YAML route to RouteFormNode
	function yamlRouteToForm(r: any): RouteFormNode {
		if (!r) return emptyNode();
		return {
			receiver: r.receiver ?? '',
			continue: r.continue ?? false,
			group_by: r.group_by ?? [],
			group_wait: r.group_wait ?? '',
			group_interval: r.group_interval ?? '',
			repeat_interval: r.repeat_interval ?? '',
			matchers: (r.matchers ?? []).map((m: any) => {
				// Matchers can be strings like 'label=value' or objects
				if (typeof m === 'string') {
					const match = m.match(/^(\w+)(=~|!=|!~|=)(.*)$/);
					if (match) {
						const [, name, op, value] = match;
						return {
							name,
							value: value.replace(/^"(.*)"$/, '$1'),
							isRegex: op === '=~' || op === '!~',
							isEqual: op === '=' || op === '=~'
						};
					}
					return { name: m, value: '', isRegex: false, isEqual: true };
				}
				return {
					name: m.name ?? '',
					value: m.value ?? '',
					isRegex: m.is_regex ?? false,
					isEqual: m.is_equal ?? true
				};
			}),
			mute_time_intervals: r.mute_time_intervals ?? [],
			active_time_intervals: r.active_time_intervals ?? [],
			routes: (r.routes ?? []).map(yamlRouteToForm)
		};
	}

	// SPEC-01: convert RouteFormNode back to plain object for YAML serialization
	function formRouteToYaml(r: RouteFormNode): any {
		const out: any = { receiver: r.receiver };
		if (r.continue) out.continue = true;
		if (r.group_by.length) out.group_by = r.group_by;
		if (r.group_wait) out.group_wait = r.group_wait;
		if (r.group_interval) out.group_interval = r.group_interval;
		if (r.repeat_interval) out.repeat_interval = r.repeat_interval;
		if (r.matchers.length) {
			out.matchers = r.matchers.map(m => {
				const op = m.isRegex ? (m.isEqual ? '=~' : '!~') : (m.isEqual ? '=' : '!=');
				return `${m.name}${op}"${m.value}"`;
			});
		}
		if (r.mute_time_intervals.length) out.mute_time_intervals = r.mute_time_intervals;
		if (r.active_time_intervals.length) out.active_time_intervals = r.active_time_intervals;
		if (r.routes.length) out.routes = r.routes.map(formRouteToYaml);
		return out;
	}

	function syncYamlToForm() {
		try {
			const parsed = yaml.load(rawYaml) as any;
			formRoute = yamlRouteToForm(parsed?.route);
		} catch {
			toast.error('Invalid YAML, cannot convert to form');
		}
	}

	function switchTab(tab: 'yaml' | 'form') {
		if (tab === 'form' && editorTab === 'yaml') {
			syncYamlToForm();
		} else if (tab === 'yaml' && editorTab === 'form') {
			// Copy the derived YAML into the editor so the user sees the current form state.
			rawYaml = formYaml;
		}
		editorTab = tab;
	}

	// For viewers: load the read-only routing tree without hitting the builder API.
	let readOnlyRoute = $state<RouteNode | null>(null);

	onMount(async () => {
		if ($canEditConfig) {
			load();
		} else {
			loading = true;
			try {
				const routing = await fetchRouting(selectedInstance || undefined);
				readOnlyRoute = routing?.route ?? null;
			} catch {
				// silently ignore — RoutingTree handles null
			} finally {
				loading = false;
			}
		}
	});
</script>

{#if $canEditConfig}
<div class="grid grid-cols-1 xl:grid-cols-2 gap-6 mt-4">
	<!-- Left: YAML editor or Form editor -->
	<div class="space-y-3">
		<div class="flex items-center justify-between">
			<div class="flex border-b gap-1">
				<button
					onclick={() => switchTab('yaml')}
					class="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium border-b-2 transition-colors
						{editorTab === 'yaml' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}"
				>
					<Code class="h-4 w-4" />
					YAML
				</button>
				<button
					onclick={() => switchTab('form')}
					class="flex items-center gap-1.5 px-3 py-1.5 text-sm font-medium border-b-2 transition-colors
						{editorTab === 'form' ? 'border-primary text-primary' : 'border-transparent text-muted-foreground hover:text-foreground'}"
				>
					<FormInput class="h-4 w-4" />
					Visual form
				</button>
			</div>
			<select bind:value={selectedInstance} onchange={load} class="px-2 py-1 rounded border bg-background text-sm">
				<option value="">Default</option>
				{#each $instances as inst}
					<option value={inst.name}>{inst.name}</option>
				{/each}
			</select>
		</div>

		{#if validationErrors.length}
			<div class="p-3 rounded-md bg-destructive/10 text-destructive text-sm space-y-1">
				{#each validationErrors as err}
					<div class="flex items-start gap-1">
						<AlertTriangle class="h-4 w-4 flex-shrink-0 mt-0.5" />
						{err}
					</div>
				{/each}
			</div>
		{/if}

		{#if editorTab === 'yaml'}
			<textarea
				bind:value={rawYaml}
				class="w-full h-96 px-3 py-2 rounded-lg border bg-card font-mono text-xs resize-none focus:outline-none focus:ring-2 focus:ring-ring"
				placeholder="global:\n  resolve_timeout: 5m\n..."
			></textarea>
		{:else}
			<div class="h-96 overflow-y-auto border rounded-lg p-2">
				<RouteNodeEditor
					route={formRoute}
					onUpdate={(r) => { formRoute = r; }}
					isRoot={true}
					availableTimeIntervals={availableTimeIntervals}
					availableReceivers={availableReceivers}
				/>
			</div>
		{/if}

		<div class="flex gap-2">
			<button onclick={previewDiff} class="flex items-center gap-2 px-4 py-2 rounded-md bg-secondary text-secondary-foreground hover:bg-secondary/80 text-sm transition-colors">
				<Eye class="h-4 w-4" />
				Preview diff
			</button>
		</div>
	</div>

	<!-- Right: Routing tree preview or diff -->
	<div class="space-y-3">
		{#if step === 'diff' && diffResult}
			<div class="space-y-4">
				<h2 class="font-semibold">Diff</h2>
				<YamlDiffViewer diff={diffResult.diff} hasChanges={diffResult.has_changes} />

				{#if diffResult.has_changes}
					<!-- Save options -->
					<div class="p-4 rounded-lg border space-y-3">
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

					<div class="flex gap-2">
						<button onclick={() => step = 'edit'} class="px-4 py-2 rounded-md border text-sm hover:bg-muted transition-colors">
							Back
						</button>
						<button onclick={save} disabled={saving} class="flex items-center gap-2 px-4 py-2 rounded-md bg-primary text-primary-foreground hover:bg-primary/90 text-sm disabled:opacity-50 transition-colors">
							<Save class="h-4 w-4" />
							{saving ? 'Saving...' : 'Confirm and save'}
						</button>
					</div>
				{/if}
			</div>
		{:else if editorTab === 'form'}
			<!-- Live YAML preview (AC-6): derived from form state, always up-to-date. -->
			<h2 class="font-semibold text-sm text-muted-foreground">Live YAML preview</h2>
			<pre class="h-96 overflow-y-auto rounded-lg border bg-card px-3 py-2 font-mono text-xs leading-relaxed whitespace-pre-wrap">{formYaml}</pre>
		{:else if routeData}
			<h2 class="font-semibold">Routing tree preview</h2>
			<RoutingTree route={routeData.route} />
		{:else if loading}
			<div class="py-12 text-center text-muted-foreground animate-pulse">Loading…</div>
		{/if}
	</div>
</div>
{:else}
<!-- Read-only view for viewers and silencers (AC-7) -->
<div class="mt-4 space-y-4">
	<div class="flex items-center gap-2 p-3 rounded-lg border bg-muted/30 text-sm text-muted-foreground">
		<Lock class="h-4 w-4 flex-shrink-0" />
		You need the <strong class="mx-1 text-foreground">config-editor</strong> role to edit routing rules.
	</div>
	{#if loading}
		<div class="py-12 text-center text-muted-foreground animate-pulse">Loading…</div>
	{:else if readOnlyRoute}
		<RoutingTree route={readOnlyRoute} />
	{/if}
</div>
{/if}
