<script lang="ts">
	let { diff, hasChanges }: { diff: string; hasChanges: boolean } = $props();
</script>

{#if !hasChanges}
	<div class="p-4 rounded-lg bg-muted text-sm text-muted-foreground text-center">
		Aucune modification détectée.
	</div>
{:else}
	<div class="rounded-lg border overflow-auto max-h-96 bg-card">
		<pre class="p-4 text-xs font-mono whitespace-pre-wrap">{#each diff.split('\n') as line}
{#if line.startsWith('+')}
<span class="block bg-green-100 dark:bg-green-900/30 text-green-800 dark:text-green-300">{line}</span>
{:else if line.startsWith('-')}
<span class="block bg-red-100 dark:bg-red-900/30 text-red-800 dark:text-red-300">{line}</span>
{:else}
<span class="block">{line}</span>
{/if}
{/each}</pre>
	</div>
{/if}
