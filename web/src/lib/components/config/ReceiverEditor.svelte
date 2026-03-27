<script lang="ts">
	import { Plus, Trash2, AlertTriangle } from 'lucide-svelte';
	import type {
		ReceiverDef,
		WebhookConfigDef,
		SlackConfigDef,
		EmailConfigDef,
		PagerdutyConfigDef,
		OpsgenieConfigDef
	} from '$lib/api/types';

	type IntegrationType = 'webhook' | 'slack' | 'pagerduty' | 'email' | 'opsgenie';

	let {
		receiver,
		onUpdate,
		validationErrors = [],
		readonly = false
	}: {
		receiver: ReceiverDef;
		onUpdate: (r: ReceiverDef) => void;
		validationErrors?: string[];
		readonly?: boolean;
	} = $props();

	const INTEGRATION_OPTIONS: { value: IntegrationType; label: string }[] = [
		{ value: 'webhook', label: 'Webhook' },
		{ value: 'slack', label: 'Slack' },
		{ value: 'pagerduty', label: 'PagerDuty' },
		{ value: 'email', label: 'Email' },
		{ value: 'opsgenie', label: 'OpsGenie' }
	];

	const TYPE_BADGE_COLORS: Record<IntegrationType, string> = {
		webhook: 'bg-blue-500/10 text-blue-700 dark:text-blue-400 border-blue-500/30',
		slack: 'bg-purple-500/10 text-purple-700 dark:text-purple-400 border-purple-500/30',
		pagerduty: 'bg-green-500/10 text-green-700 dark:text-green-400 border-green-500/30',
		email: 'bg-yellow-500/10 text-yellow-700 dark:text-yellow-400 border-yellow-500/30',
		opsgenie: 'bg-orange-500/10 text-orange-700 dark:text-orange-400 border-orange-500/30'
	};

	function addIntegration(type: IntegrationType) {
		switch (type) {
			case 'webhook':
				onUpdate({
					...receiver,
					webhook_configs: [...(receiver.webhook_configs ?? []), { url: '', send_resolved: true }]
				});
				break;
			case 'slack':
				onUpdate({
					...receiver,
					slack_configs: [...(receiver.slack_configs ?? []), { channel: '', send_resolved: true }]
				});
				break;
			case 'pagerduty':
				onUpdate({
					...receiver,
					pagerduty_configs: [
						...(receiver.pagerduty_configs ?? []),
						{ routing_key: '', send_resolved: true }
					]
				});
				break;
			case 'email':
				onUpdate({
					...receiver,
					email_configs: [...(receiver.email_configs ?? []), { to: '', send_resolved: true }]
				});
				break;
			case 'opsgenie':
				onUpdate({
					...receiver,
					opsgenie_configs: [
						...(receiver.opsgenie_configs ?? []),
						{ api_key: '', send_resolved: true }
					]
				});
				break;
		}
	}

	function removeWebhook(i: number) {
		onUpdate({
			...receiver,
			webhook_configs: (receiver.webhook_configs ?? []).filter((_, idx) => idx !== i)
		});
	}

	function patchWebhook(i: number, patch: Partial<WebhookConfigDef>) {
		onUpdate({
			...receiver,
			webhook_configs: (receiver.webhook_configs ?? []).map((c, idx) =>
				idx === i ? { ...c, ...patch } : c
			)
		});
	}

	function removeSlack(i: number) {
		onUpdate({
			...receiver,
			slack_configs: (receiver.slack_configs ?? []).filter((_, idx) => idx !== i)
		});
	}

	function patchSlack(i: number, patch: Partial<SlackConfigDef>) {
		onUpdate({
			...receiver,
			slack_configs: (receiver.slack_configs ?? []).map((c, idx) =>
				idx === i ? { ...c, ...patch } : c
			)
		});
	}

	function removePagerduty(i: number) {
		onUpdate({
			...receiver,
			pagerduty_configs: (receiver.pagerduty_configs ?? []).filter((_, idx) => idx !== i)
		});
	}

	function patchPagerduty(i: number, patch: Partial<PagerdutyConfigDef>) {
		onUpdate({
			...receiver,
			pagerduty_configs: (receiver.pagerduty_configs ?? []).map((c, idx) =>
				idx === i ? { ...c, ...patch } : c
			)
		});
	}

	function removeEmail(i: number) {
		onUpdate({
			...receiver,
			email_configs: (receiver.email_configs ?? []).filter((_, idx) => idx !== i)
		});
	}

	function patchEmail(i: number, patch: Partial<EmailConfigDef>) {
		onUpdate({
			...receiver,
			email_configs: (receiver.email_configs ?? []).map((c, idx) =>
				idx === i ? { ...c, ...patch } : c
			)
		});
	}

	function removeOpsgenie(i: number) {
		onUpdate({
			...receiver,
			opsgenie_configs: (receiver.opsgenie_configs ?? []).filter((_, idx) => idx !== i)
		});
	}

	function patchOpsgenie(i: number, patch: Partial<OpsgenieConfigDef>) {
		onUpdate({
			...receiver,
			opsgenie_configs: (receiver.opsgenie_configs ?? []).map((c, idx) =>
				idx === i ? { ...c, ...patch } : c
			)
		});
	}
</script>

{#if receiver.raw_yaml}
	<!-- Raw YAML mode: unknown integration type, edit as textarea -->
	<div class="space-y-2">
		<div class="flex items-center gap-2">
			<span class="text-xs font-medium text-muted-foreground">Raw YAML</span>
			<span class="px-2 py-0.5 rounded-full text-xs border bg-muted/50 text-muted-foreground">
				Unknown integration type — editing as raw YAML
			</span>
		</div>
		<textarea
			value={receiver.raw_yaml}
			oninput={(e) => onUpdate({ ...receiver, raw_yaml: (e.target as HTMLTextAreaElement).value })}
			rows={10}
			disabled={readonly}
			class="w-full px-3 py-2 rounded border bg-background font-mono text-xs resize-y disabled:opacity-60 disabled:cursor-not-allowed"
		></textarea>
	</div>
{:else}
	<!-- Form mode: typed integration editor -->
	<div class="space-y-3">
		<!-- Webhook configs -->
		{#each receiver.webhook_configs ?? [] as cfg, i}
			<div class="rounded-lg border bg-card overflow-hidden">
				<div class="flex items-center gap-2 px-3 py-2 bg-muted/30 border-b">
					<span class="px-2 py-0.5 rounded-full text-xs border {TYPE_BADGE_COLORS.webhook}">Webhook</span>
					<span class="flex-1 text-xs text-muted-foreground truncate">{cfg.url || 'new'}</span>
					{#if !readonly}
						<button
							onclick={() => removeWebhook(i)}
							class="p-0.5 rounded text-muted-foreground hover:text-destructive transition-colors"
							title="Remove"
						>
							<Trash2 class="h-3 w-3" />
						</button>
					{/if}
				</div>
				{#if !readonly}
					<div class="p-3 space-y-2">
						<div>
							<span class="text-xs text-muted-foreground">URL <span class="text-destructive">*</span></span>
							<input
								value={cfg.url}
								oninput={(e) => patchWebhook(i, { url: (e.target as HTMLInputElement).value })}
								placeholder="https://example.com/webhook"
								class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
							/>
						</div>
						<div>
							<span class="text-xs text-muted-foreground">Max alerts (0 = unlimited)</span>
							<input
								type="number"
								value={cfg.max_alerts ?? 0}
								oninput={(e) => patchWebhook(i, { max_alerts: parseInt((e.target as HTMLInputElement).value) || 0 })}
								min="0"
								class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
							/>
						</div>
						<label class="flex items-center gap-2 text-xs cursor-pointer">
							<input
								type="checkbox"
								checked={cfg.send_resolved ?? true}
								onchange={(e) => patchWebhook(i, { send_resolved: (e.target as HTMLInputElement).checked })}
							/>
							Send resolved notifications
						</label>
					</div>
				{/if}
			</div>
		{/each}

		<!-- Slack configs -->
		{#each receiver.slack_configs ?? [] as cfg, i}
			<div class="rounded-lg border bg-card overflow-hidden">
				<div class="flex items-center gap-2 px-3 py-2 bg-muted/30 border-b">
					<span class="px-2 py-0.5 rounded-full text-xs border {TYPE_BADGE_COLORS.slack}">Slack</span>
					<span class="flex-1 text-xs text-muted-foreground truncate">{cfg.channel || 'new'}</span>
					{#if !readonly}
						<button
							onclick={() => removeSlack(i)}
							class="p-0.5 rounded text-muted-foreground hover:text-destructive transition-colors"
							title="Remove"
						>
							<Trash2 class="h-3 w-3" />
						</button>
					{/if}
				</div>
				{#if !readonly}
					<div class="p-3 space-y-2">
						<div>
							<span class="text-xs text-muted-foreground">Channel <span class="text-destructive">*</span></span>
							<input
								value={cfg.channel}
								oninput={(e) => patchSlack(i, { channel: (e.target as HTMLInputElement).value })}
								placeholder="#alerts"
								class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
							/>
						</div>
						<div class="grid grid-cols-2 gap-2">
							<div>
								<span class="text-xs text-muted-foreground">API URL</span>
								<input
									value={cfg.api_url ?? ''}
									oninput={(e) => patchSlack(i, { api_url: (e.target as HTMLInputElement).value || undefined })}
									placeholder="https://hooks.slack.com/..."
									class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
								/>
							</div>
							<div>
								<span class="text-xs text-muted-foreground">Username</span>
								<input
									value={cfg.username ?? ''}
									oninput={(e) => patchSlack(i, { username: (e.target as HTMLInputElement).value || undefined })}
									placeholder="alertmanager"
									class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
								/>
							</div>
						</div>
						<div>
							<span class="text-xs text-muted-foreground">Title</span>
							<input
								value={cfg.title ?? ''}
								oninput={(e) => patchSlack(i, { title: (e.target as HTMLInputElement).value || undefined })}
								placeholder={"[{{ .Status | toUpper }}] {{ .CommonLabels.alertname }}"}
								class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
							/>
						</div>
						<div>
							<span class="text-xs text-muted-foreground">Text</span>
							<input
								value={cfg.text ?? ''}
								oninput={(e) => patchSlack(i, { text: (e.target as HTMLInputElement).value || undefined })}
								placeholder="Message body"
								class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
							/>
						</div>
						<label class="flex items-center gap-2 text-xs cursor-pointer">
							<input
								type="checkbox"
								checked={cfg.send_resolved ?? true}
								onchange={(e) => patchSlack(i, { send_resolved: (e.target as HTMLInputElement).checked })}
							/>
							Send resolved notifications
						</label>
					</div>
				{/if}
			</div>
		{/each}

		<!-- PagerDuty configs -->
		{#each receiver.pagerduty_configs ?? [] as cfg, i}
			<div class="rounded-lg border bg-card overflow-hidden">
				<div class="flex items-center gap-2 px-3 py-2 bg-muted/30 border-b">
					<span class="px-2 py-0.5 rounded-full text-xs border {TYPE_BADGE_COLORS.pagerduty}">PagerDuty</span>
					<span class="flex-1 text-xs text-muted-foreground truncate">{cfg.routing_key || cfg.service_key || 'new'}</span>
					{#if !readonly}
						<button
							onclick={() => removePagerduty(i)}
							class="p-0.5 rounded text-muted-foreground hover:text-destructive transition-colors"
							title="Remove"
						>
							<Trash2 class="h-3 w-3" />
						</button>
					{/if}
				</div>
				{#if !readonly}
					<div class="p-3 space-y-2">
						<div class="grid grid-cols-2 gap-2">
							<div>
								<span class="text-xs text-muted-foreground">Routing key</span>
								<input
									value={cfg.routing_key ?? ''}
									oninput={(e) => patchPagerduty(i, { routing_key: (e.target as HTMLInputElement).value || undefined })}
									placeholder="(v2 integration)"
									class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
								/>
							</div>
							<div>
								<span class="text-xs text-muted-foreground">Service key</span>
								<input
									value={cfg.service_key ?? ''}
									oninput={(e) => patchPagerduty(i, { service_key: (e.target as HTMLInputElement).value || undefined })}
									placeholder="(v1 integration)"
									class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
								/>
							</div>
						</div>
						<div>
							<span class="text-xs text-muted-foreground">Description</span>
							<input
								value={cfg.description ?? ''}
								oninput={(e) => patchPagerduty(i, { description: (e.target as HTMLInputElement).value || undefined })}
								placeholder={"{{ .CommonAnnotations.summary }}"}
								class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
							/>
						</div>
						<label class="flex items-center gap-2 text-xs cursor-pointer">
							<input
								type="checkbox"
								checked={cfg.send_resolved ?? true}
								onchange={(e) => patchPagerduty(i, { send_resolved: (e.target as HTMLInputElement).checked })}
							/>
							Send resolved notifications
						</label>
					</div>
				{/if}
			</div>
		{/each}

		<!-- Email configs -->
		{#each receiver.email_configs ?? [] as cfg, i}
			<div class="rounded-lg border bg-card overflow-hidden">
				<div class="flex items-center gap-2 px-3 py-2 bg-muted/30 border-b">
					<span class="px-2 py-0.5 rounded-full text-xs border {TYPE_BADGE_COLORS.email}">Email</span>
					<span class="flex-1 text-xs text-muted-foreground truncate">{cfg.to || 'new'}</span>
					{#if !readonly}
						<button
							onclick={() => removeEmail(i)}
							class="p-0.5 rounded text-muted-foreground hover:text-destructive transition-colors"
							title="Remove"
						>
							<Trash2 class="h-3 w-3" />
						</button>
					{/if}
				</div>
				{#if !readonly}
					<div class="p-3 space-y-2">
						<div>
							<span class="text-xs text-muted-foreground">To <span class="text-destructive">*</span></span>
							<input
								value={cfg.to}
								oninput={(e) => patchEmail(i, { to: (e.target as HTMLInputElement).value })}
								placeholder="oncall@example.com"
								class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
							/>
						</div>
						<div class="grid grid-cols-2 gap-2">
							<div>
								<span class="text-xs text-muted-foreground">From</span>
								<input
									value={cfg.from ?? ''}
									oninput={(e) => patchEmail(i, { from: (e.target as HTMLInputElement).value || undefined })}
									placeholder="alertmanager@example.com"
									class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
								/>
							</div>
							<div>
								<span class="text-xs text-muted-foreground">Smarthost</span>
								<input
									value={cfg.smarthost ?? ''}
									oninput={(e) => patchEmail(i, { smarthost: (e.target as HTMLInputElement).value || undefined })}
									placeholder="smtp.example.com:587"
									class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
								/>
							</div>
						</div>
						<div class="grid grid-cols-2 gap-2">
							<div>
								<span class="text-xs text-muted-foreground">SMTP username</span>
								<input
									value={cfg.auth_username ?? ''}
									oninput={(e) => patchEmail(i, { auth_username: (e.target as HTMLInputElement).value || undefined })}
									placeholder="user"
									class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
								/>
							</div>
							<div>
								<span class="text-xs text-muted-foreground">SMTP password</span>
								<input
									type="password"
									value={cfg.auth_password ?? ''}
									oninput={(e) => patchEmail(i, { auth_password: (e.target as HTMLInputElement).value || undefined })}
									placeholder="••••••••"
									class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
								/>
							</div>
						</div>
						<label class="flex items-center gap-2 text-xs cursor-pointer">
							<input
								type="checkbox"
								checked={cfg.send_resolved ?? true}
								onchange={(e) => patchEmail(i, { send_resolved: (e.target as HTMLInputElement).checked })}
							/>
							Send resolved notifications
						</label>
					</div>
				{/if}
			</div>
		{/each}

		<!-- OpsGenie configs -->
		{#each receiver.opsgenie_configs ?? [] as cfg, i}
			<div class="rounded-lg border bg-card overflow-hidden">
				<div class="flex items-center gap-2 px-3 py-2 bg-muted/30 border-b">
					<span class="px-2 py-0.5 rounded-full text-xs border {TYPE_BADGE_COLORS.opsgenie}">OpsGenie</span>
					<span class="flex-1 text-xs text-muted-foreground truncate">{cfg.message || 'new'}</span>
					{#if !readonly}
						<button
							onclick={() => removeOpsgenie(i)}
							class="p-0.5 rounded text-muted-foreground hover:text-destructive transition-colors"
							title="Remove"
						>
							<Trash2 class="h-3 w-3" />
						</button>
					{/if}
				</div>
				{#if !readonly}
					<div class="p-3 space-y-2">
						<div>
							<span class="text-xs text-muted-foreground">API key</span>
							<input
								type="password"
								value={cfg.api_key ?? ''}
								oninput={(e) => patchOpsgenie(i, { api_key: (e.target as HTMLInputElement).value || undefined })}
								placeholder="••••••••"
								class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
							/>
						</div>
						<div class="grid grid-cols-2 gap-2">
							<div>
								<span class="text-xs text-muted-foreground">Message</span>
								<input
									value={cfg.message ?? ''}
									oninput={(e) => patchOpsgenie(i, { message: (e.target as HTMLInputElement).value || undefined })}
									placeholder={"{{ .CommonLabels.alertname }}"}
									class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
								/>
							</div>
							<div>
								<span class="text-xs text-muted-foreground">Priority</span>
								<input
									value={cfg.priority ?? ''}
									oninput={(e) => patchOpsgenie(i, { priority: (e.target as HTMLInputElement).value || undefined })}
									placeholder="P1"
									class="w-full px-2 py-1 rounded border bg-background text-xs mt-0.5"
								/>
							</div>
						</div>
						<label class="flex items-center gap-2 text-xs cursor-pointer">
							<input
								type="checkbox"
								checked={cfg.send_resolved ?? true}
								onchange={(e) => patchOpsgenie(i, { send_resolved: (e.target as HTMLInputElement).checked })}
							/>
							Send resolved notifications
						</label>
					</div>
				{/if}
			</div>
		{/each}

		<!-- Add integration dropdown -->
		{#if !readonly}
			<div class="flex items-center gap-2">
				<select
					onchange={(e) => {
						const val = (e.target as HTMLSelectElement).value as IntegrationType;
						if (val) {
							addIntegration(val);
							(e.target as HTMLSelectElement).value = '';
						}
					}}
					class="px-2 py-1 rounded border bg-background text-xs"
				>
					<option value="">+ Add integration</option>
					{#each INTEGRATION_OPTIONS as opt}
						<option value={opt.value}>{opt.label}</option>
					{/each}
				</select>
			</div>
		{/if}
	</div>
{/if}

<!-- Validation errors -->
{#if validationErrors.length > 0}
	<div class="mt-3 rounded-md border border-destructive/50 bg-destructive/10 p-3">
		<div class="flex items-start gap-2">
			<AlertTriangle class="h-4 w-4 text-destructive mt-0.5 shrink-0" />
			<ul class="space-y-0.5">
				{#each validationErrors as err}
					<li class="text-xs text-destructive">{err}</li>
				{/each}
			</ul>
		</div>
	</div>
{/if}
