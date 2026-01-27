<script lang="ts">
	import { ChevronRight, ChevronDown } from '@lucide/svelte';

	let { data, defaultExpandLevel = 1 } = $props<{ data: any; defaultExpandLevel?: number }>();

	let parsedData = $state<any>(null);
	let parseError = $state<string | null>(null);
	let expandedPaths = $state<Set<string>>(new Set());

	$effect(() => {
		parseError = null;
		expandedPaths = new Set();
		if (data === null || data === undefined) {
			parsedData = data;
		} else if (typeof data === 'string') {
			try {
				parsedData = JSON.parse(data);
			} catch (e) {
				parseError = e instanceof Error ? e.message : 'Failed to parse JSON';
				parsedData = null;
			}
		} else {
			parsedData = data;
		}
	});

	function getType(value: any): string {
		if (value === null) return 'null';
		if (Array.isArray(value)) return 'array';
		return typeof value;
	}

	function getItemCount(value: any): number {
		if (Array.isArray(value)) return value.length;
		if (typeof value === 'object' && value !== null) return Object.keys(value).length;
		return 0;
	}

	function isExpanded(path: string, depth: number): boolean {
		if (expandedPaths.has(path)) return true;
		if (expandedPaths.has(`!${path}`)) return false;
		// Default: expand if depth < defaultExpandLevel
		return depth < defaultExpandLevel;
	}

	function toggleExpanded(path: string, depth: number) {
		const currentlyExpanded = isExpanded(path, depth);
		if (currentlyExpanded) {
			// Collapse: mark as explicitly collapsed
			expandedPaths.delete(path);
			expandedPaths.add(`!${path}`);
		} else {
			// Expand: mark as explicitly expanded
			expandedPaths.delete(`!${path}`);
			expandedPaths.add(path);
		}
		expandedPaths = new Set(expandedPaths);
	}
</script>

{#snippet jsonNode(
	value: any,
	key: string | number | null,
	depth: number,
	isLast: boolean,
	path: string
)}
	{@const type = getType(value)}
	{@const isExpandable = type === 'object' || type === 'array'}
	{@const itemCount = getItemCount(value)}
	{@const expanded = isExpanded(path, depth)}

	{#if isExpandable}
		<div class="leading-6">
			<button
				type="button"
				class="-ml-1 inline-flex cursor-pointer items-center rounded px-1 hover:bg-muted/50"
				onclick={() => toggleExpanded(path, depth)}
			>
				{#if expanded}
					<ChevronDown class="h-3 w-3 shrink-0 text-muted-foreground" />
				{:else}
					<ChevronRight class="h-3 w-3 shrink-0 text-muted-foreground" />
				{/if}
				{#if key !== null}
					<span class="text-foreground">"{key}"</span><span class="text-muted-foreground">: </span>
				{/if}
				{#if expanded}
					<span class="text-muted-foreground">{type === 'array' ? '[' : '{'}</span>
				{:else}
					<span class="text-muted-foreground">
						{type === 'array' ? `[${itemCount} items]` : `{${itemCount} items}`}
					</span>
				{/if}
			</button>
			{#if expanded}
				<div class="ml-1.5 border-l border-border pl-4">
					{#if type === 'array'}
						{#each value as item, i}
							{@render jsonNode(item, i, depth + 1, i === value.length - 1, `${path}[${i}]`)}
						{/each}
					{:else}
						{@const entries = Object.entries(value)}
						{#each entries as [k, v], i}
							{@render jsonNode(v, k, depth + 1, i === entries.length - 1, `${path}.${k}`)}
						{/each}
					{/if}
				</div>
				<span class="pl-1 text-muted-foreground"
					>{type === 'array' ? ']' : '}'}{isLast ? '' : ','}</span
				>
			{/if}
		</div>
	{:else}
		<div class="pl-4 leading-6">
			{#if key !== null}
				<span class="text-foreground">"{key}"</span><span class="text-muted-foreground">: </span>
			{/if}
			{#if type === 'string'}
				<span class="text-green-600 dark:text-green-400">"{value}"</span>
			{:else if type === 'number'}
				<span class="text-blue-600 dark:text-blue-400">{value}</span>
			{:else if type === 'boolean'}
				<span class="text-purple-600 dark:text-purple-400">{value ? 'true' : 'false'}</span>
			{:else if type === 'null'}
				<span class="text-muted-foreground">null</span>
			{:else}
				<span class="text-foreground">{String(value)}</span>
			{/if}
			{#if !isLast}
				<span class="text-muted-foreground">,</span>
			{/if}
		</div>
	{/if}
{/snippet}

<div class="max-h-[60vh] overflow-auto rounded border bg-muted/30 p-4 font-mono text-sm">
	{#if parseError}
		<div class="text-destructive">
			<span class="font-semibold">Invalid JSON:</span>
			{parseError}
		</div>
	{:else if parsedData === null}
		<span class="text-muted-foreground">null</span>
	{:else if parsedData === undefined}
		<span class="text-muted-foreground">undefined</span>
	{:else}
		{@render jsonNode(parsedData, null, 0, true, '$')}
	{/if}
</div>
