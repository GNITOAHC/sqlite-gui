<script lang="ts">
	import { api } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Table from '$lib/components/ui/table/index.js';
	import hljs from 'highlight.js/lib/core';
	import sqlLang from 'highlight.js/lib/languages/sql';
	import 'highlight.js/styles/github-dark.css';

	hljs.registerLanguage('sql', sqlLang);

	let { db } = $props<{ db: string }>();

	let sql = $state('');
	let result = $state<any>(null);
	let error = $state<string | null>(null);
	let loading = $state(false);
	let isExec = $state(false);

	let highlightedCode = $derived(hljs.highlight(sql || ' ', { language: 'sql' }).value);

	async function runQuery() {
		if (!sql.trim()) return;
		loading = true;
		error = null;
		result = null;

		// Simple heuristic
		const upper = sql.trim().toUpperCase();
		const isSelect = /^\s*(SELECT|PRAGMA|WITH|VALUES|EXPLAIN)\b/.test(upper);
		isExec = !isSelect;

		try {
			if (isSelect) {
				const res = await api<any, any>(`/query?db=${db}`, {
					method: 'POST',
					body: { query: sql, args: [] }
				});
				result = res;
			} else {
				const res = await api<any, any>(`/exec?db=${db}`, {
					method: 'POST',
					body: { query: sql, args: [] }
				});
				result = res;
			}
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	function onKeydown(e: KeyboardEvent) {
		if ((e.ctrlKey || e.metaKey) && e.key === 'Enter') {
			e.preventDefault();
			runQuery();
		}
	}
</script>

<div class="space-y-4">
	<div class="relative grid min-h-[150px] rounded-md border bg-muted/50 font-mono text-sm">
		<!-- Highlighter -->
		<pre
			aria-hidden="true"
			class="wrap-break-words pointer-events-none col-start-1 row-start-1 m-0 p-4 font-mono text-sm leading-relaxed whitespace-pre-wrap"><code
				class="language-sql">{@html highlightedCode}</code
			><span class="invisible">{' '}</span></pre>

		<!-- Input -->
		<textarea
			class="col-start-1 row-start-1 m-0 h-full w-full resize-none overflow-hidden bg-transparent p-4 font-mono text-sm leading-relaxed text-transparent ring-0 outline-none placeholder:text-muted-foreground/50"
			style="caret-color: var(--foreground);"
			bind:value={sql}
			onkeydown={onKeydown}
			placeholder="SELECT * FROM table..."
			spellcheck="false"
		></textarea>
	</div>

	<div class="flex justify-end">
		<Button onclick={runQuery} disabled={loading}>
			{#if loading}Running...{:else}Run (Cmd+Enter){/if}
		</Button>
	</div>

	{#if error}
		<div class="rounded-md bg-destructive/10 p-4 text-destructive">
			Error: {error}
		</div>
	{/if}

	{#if result}
		{#if !isExec}
			<!-- Query Result -->
			{#if result.rows && result.rows.length > 0}
				<div class="overflow-x-auto rounded-md border">
					<Table.Root>
						<Table.Header>
							<Table.Row>
								{#each Object.keys(result.rows[0]) as col}
									<Table.Head>{col}</Table.Head>
								{/each}
							</Table.Row>
						</Table.Header>
						<Table.Body>
							{#each result.rows as row}
								<Table.Row>
									{#each Object.keys(result.rows[0]) as col}
										<Table.Cell class="whitespace-nowrap">{row[col]}</Table.Cell>
									{/each}
								</Table.Row>
							{/each}
						</Table.Body>
					</Table.Root>
				</div>
			{:else}
				<p class="text-muted-foreground">No results found.</p>
			{/if}
		{:else}
			<!-- Exec Result -->
			<div class="rounded-md border p-4">
				<p>Rows Affected: {result.rowsAffected}</p>
				<p>Last Insert ID: {result.lastInsertId}</p>
			</div>
		{/if}
	{/if}
</div>

<style>
	textarea,
	pre {
		font-family:
			ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New',
			monospace;
	}

	:global(.hljs) {
		background: transparent !important;
		padding: 0 !important;
	}
</style>
