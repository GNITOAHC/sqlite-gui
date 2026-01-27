<script lang="ts">
	import { api } from '$lib/api/client';
	import { onMount } from 'svelte';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { EditorView, keymap, placeholder, drawSelection } from '@codemirror/view';
	import { EditorState, Compartment } from '@codemirror/state';
	import { sql as sqlLang } from '@codemirror/lang-sql';
	import { defaultKeymap, history, historyKeymap } from '@codemirror/commands';
	import { syntaxHighlighting, defaultHighlightStyle, bracketMatching } from '@codemirror/language';
	import {
		autocompletion,
		completionKeymap,
		closeBrackets,
		closeBracketsKeymap
	} from '@codemirror/autocomplete';
	import ChartDisplay from './ChartDisplay.svelte';

	type ChartType = 'line' | 'bar';
	type RowLimit = number | 'unlimited';
	type Row = Record<string, unknown>;

	interface Connection {
		name: string;
		connString: string;
	}

	// State
	let connections = $state<Connection[]>([]);
	let selectedDb = $state<string>('');
	let sql = $state<string>('');
	let chartType = $state<ChartType>('line');
	let rowLimit = $state<number>(100);
	let isUnlimited = $state<boolean>(false);
	let loading = $state<boolean>(false);
	let error = $state<string | null>(null);
	let queryResult = $state<Row[] | null>(null);

	// CodeMirror
	let editorContainer: HTMLDivElement;
	let editorView: EditorView;
	const schemaConfig = new Compartment();

	// Derived values
	const effectiveLimit = $derived<RowLimit>(isUnlimited ? 'unlimited' : rowLimit);

	// Build query with LIMIT
	function buildQuery(query: string, limit: RowLimit): string {
		const trimmedSql = query.trim().replace(/;$/, '');

		if (limit === 'unlimited') {
			return trimmedSql;
		}

		// Check if query already has LIMIT clause
		const hasLimit = /\bLIMIT\s+\d+/i.test(trimmedSql);
		if (hasLimit) {
			return trimmedSql;
		}

		return `${trimmedSql} LIMIT ${limit}`;
	}

	// Check if query is a SELECT-type query
	function isSelectQuery(query: string): boolean {
		return /^\s*(SELECT|PRAGMA|WITH|VALUES|EXPLAIN)\b/i.test(query.trim());
	}

	// Run query
	async function runQuery() {
		if (!sql.trim()) {
			error = 'Please enter a SQL query.';
			return;
		}

		if (!selectedDb) {
			error = 'Please select a database.';
			return;
		}

		if (!isSelectQuery(sql)) {
			error =
				'Only SELECT queries are supported for visualization. Please use a SELECT, PRAGMA, WITH, VALUES, or EXPLAIN query.';
			return;
		}

		loading = true;
		error = null;
		queryResult = null;

		const finalQuery = buildQuery(sql, effectiveLimit);

		try {
			const result = await api<{ query: string; args: unknown[] }, { rows: Row[] }>(
				`/query?db=${selectedDb}`,
				{
					method: 'POST',
					body: { query: finalQuery, args: [] }
				}
			);

			if (!result.rows || result.rows.length === 0) {
				error = 'No data returned. Please check your SQL query and ensure the table contains data.';
				return;
			}

			const columns = Object.keys(result.rows[0]);
			if (columns.length < 2) {
				error =
					'Chart requires at least 2 columns. Please modify your query to include both a label column (X-axis) and a value column (Y-axis).';
				return;
			}

			queryResult = result.rows;
		} catch (e: unknown) {
			error = e instanceof Error ? e.message : 'An unknown error occurred.';
		} finally {
			loading = false;
		}
	}

	// Update schema for autocomplete
	async function updateSchema() {
		if (!selectedDb || !editorView) return;
		try {
			const { tables = [] } = await api<unknown, { tables: string[] }>(`/tables?db=${selectedDb}`);
			const schema: Record<string, string[]> = {};

			await Promise.all(
				tables.map(async (table) => {
					try {
						const { columns = [] } = await api<unknown, { columns: { Name: string }[] }>(
							`/tables/${table}/columns?db=${selectedDb}`
						);
						schema[table] = columns.map((c) => c.Name);
					} catch {
						// Ignore errors for individual tables
					}
				})
			);

			editorView.dispatch({ effects: schemaConfig.reconfigure(sqlLang({ schema })) });
		} catch {
			// Ignore schema errors
		}
	}

	// Effect to update schema when database changes
	$effect(() => {
		if (editorView && selectedDb) {
			updateSchema();
		}
	});

	// Fetch connections on mount
	onMount(async () => {
		try {
			const result = await api<unknown, { connections: Connection[] }>('/connections');
			connections = result.connections ?? [];
			if (connections.length > 0) {
				selectedDb = connections[0].name;
			}
		} catch (e) {
			console.error('Failed to load connections:', e);
		}
	});

	// Initialize CodeMirror
	onMount(() => {
		if (!editorContainer) return;

		const extensions = [
			keymap.of([
				{
					key: 'Mod-Enter',
					run: () => {
						runQuery();
						return true;
					}
				},
				...defaultKeymap,
				...historyKeymap,
				...completionKeymap,
				...closeBracketsKeymap
			]),
			history(),
			drawSelection(),
			bracketMatching(),
			closeBrackets(),
			autocompletion(),
			syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
			schemaConfig.of(sqlLang()),
			placeholder('SELECT id, value FROM table ORDER BY id...'),
			EditorView.updateListener.of((u) => {
				if (u.docChanged) sql = u.state.doc.toString();
			}),
			EditorView.theme({
				'&': { height: '100%', backgroundColor: 'transparent', fontSize: '0.875rem' },
				'.cm-content': {
					fontFamily:
						"ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace",
					padding: '1rem'
				},
				'.cm-scroller': { fontFamily: 'inherit' },
				'&.cm-focused': { outline: 'none' }
			})
		];

		editorView = new EditorView({
			state: EditorState.create({ doc: sql, extensions }),
			parent: editorContainer
		});

		return () => editorView?.destroy();
	});
</script>

<div class="space-y-6 py-6">
	<h1 class="text-2xl font-bold">Data Visualization</h1>

	<!-- Query Form -->
	<div class="space-y-4">
		<!-- Database Selector -->
		<div class="space-y-2">
			<Label for="database">Database</Label>
			<select
				id="database"
				class="h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50"
				bind:value={selectedDb}
			>
				{#if connections.length === 0}
					<option value="" disabled>Loading connections...</option>
				{:else}
					{#each connections as conn}
						<option value={conn.name}>{conn.name}</option>
					{/each}
				{/if}
			</select>
		</div>

		<!-- SQL Query Editor -->
		<div class="space-y-2">
			<Label for="sql-editor">SQL Query</Label>
			<div
				bind:this={editorContainer}
				class="relative min-h-[120px] overflow-hidden rounded-md border bg-muted/50 font-mono text-sm"
			></div>
		</div>

		<!-- Display Method -->
		<div class="space-y-2">
			<Label for="chart-type">Display Method</Label>
			<select
				id="chart-type"
				class="h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50"
				bind:value={chartType}
			>
				<option value="line">Line Chart</option>
				<option value="bar">Bar Chart</option>
			</select>

			<!-- Chart Requirements Note -->
			<div
				class="rounded-md border border-blue-500/20 bg-blue-500/10 px-4 py-3 text-sm text-blue-700 dark:text-blue-300"
			>
				{#if chartType === 'line'}
					<p class="font-medium">📈 Line Chart Requirements:</p>
					<ul class="mt-1 list-inside list-disc space-y-1 text-blue-600 dark:text-blue-400">
						<li>
							Return at least <strong>2 columns</strong>: one for X-axis labels, one for Y-axis
							values
						</li>
						<li>Y-axis column must contain <strong>numeric data</strong></li>
						<li>
							Use <code class="rounded bg-blue-500/20 px-1">ORDER BY</code> to ensure correct data sequence
						</li>
						<li>
							Example: <code class="rounded bg-blue-500/20 px-1"
								>SELECT date, price FROM sales ORDER BY date</code
							>
						</li>
					</ul>
				{:else}
					<p class="font-medium">📊 Bar Chart Requirements:</p>
					<ul class="mt-1 list-inside list-disc space-y-1 text-blue-600 dark:text-blue-400">
						<li>Return at least <strong>2 columns</strong>: one for categories, one for values</li>
						<li>Y-axis column must contain <strong>numeric data</strong></li>
						<li>
							Example: <code class="rounded bg-blue-500/20 px-1"
								>SELECT category, COUNT(*) as count FROM products GROUP BY category</code
							>
						</li>
					</ul>
				{/if}
			</div>
		</div>

		<!-- Row Limit -->
		<div class="space-y-2">
			<Label for="row-limit">Row Limit</Label>
			<div class="flex items-center gap-4">
				<Input
					id="row-limit"
					type="number"
					min={1}
					max={10000}
					bind:value={rowLimit}
					disabled={isUnlimited}
					class="w-32"
				/>
				<label class="flex cursor-pointer items-center gap-2 text-sm">
					<input type="checkbox" bind:checked={isUnlimited} class="h-4 w-4 rounded border-input" />
					Unlimited
				</label>
			</div>
			{#if isUnlimited}
				<p class="text-xs text-muted-foreground">
					⚠️ Warning: Fetching unlimited rows may be slow for large datasets.
				</p>
			{:else}
				<p class="text-xs text-muted-foreground">
					The query will be limited to {rowLimit} rows.
					<code class="rounded bg-muted px-1">LIMIT {rowLimit}</code> will be automatically appended
					unless your query already contains a LIMIT clause.
				</p>
			{/if}
		</div>

		<!-- Run Button -->
		<div class="flex justify-end">
			<Button onclick={runQuery} disabled={loading || !selectedDb}>
				{#if loading}
					Running...
				{:else}
					Run Query (Cmd+Enter)
				{/if}
			</Button>
		</div>
	</div>

	<!-- Error Display -->
	{#if error}
		<div
			class="rounded-md border border-destructive/20 bg-destructive/10 px-4 py-3 text-destructive"
		>
			<strong>Error:</strong>
			{error}
		</div>
	{/if}

	<!-- Results Section -->
	{#if queryResult}
		<div class="space-y-4">
			<h2 class="text-lg font-semibold">Results ({queryResult.length} rows)</h2>
			<ChartDisplay data={queryResult} {chartType} />
		</div>
	{/if}

	<!-- General Tip -->
	<div class="rounded-md border bg-muted/50 px-4 py-3 text-sm text-muted-foreground">
		<strong>💡 Tip:</strong> Ensure your query returns data suitable for visualization:
		<ul class="mt-1 list-inside list-disc">
			<li>At least 2 columns are required</li>
			<li>One column for labels/categories (X-axis)</li>
			<li>One column with numeric values (Y-axis)</li>
			<li>Non-numeric Y-axis values will be skipped</li>
		</ul>
	</div>
</div>
