<script lang="ts">
	import { api } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button/index.js';
	import * as Table from '$lib/components/ui/table/index.js';
	import { onMount } from 'svelte';
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

	let { db } = $props<{ db: string }>();

	let sql = $state('');
	let result = $state<any>(null);
	let error = $state<string | null>(null);
	let loading = $state(false);
	let isExec = $state(false);

	let editorContainer: HTMLDivElement;
	let editorView: EditorView;
	const schemaConfig = new Compartment();

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

	async function updateSchema() {
		console.log('Updating schema for db:', db);
		if (!db || !editorView) return;
		console.log('Fetching tables for db:', db);

		try {
			const tablesRes = await api<any, { tables: string[] }>(`/tables?db=${db}`);
			const tables = tablesRes.tables || [];

			const schema: Record<string, string[]> = {};

			await Promise.all(
				tables.map(async (table) => {
					try {
						const colsRes = await api<any, { columns: any[] }>(`/tables/${table}/columns?db=${db}`);
						schema[table] = (colsRes.columns || []).map((c) => c.Name);
					} catch (e) {
						console.error(`Failed to load columns for ${table}`, e);
					}
				})
			);

			editorView.dispatch({
				effects: schemaConfig.reconfigure(sqlLang({ schema }))
			});
		} catch (e) {
			console.error('Failed to load schema', e);
		}
	}

	$effect(() => {
		if (editorView && db) {
			updateSchema();
		}
	});

	onMount(() => {
		if (!editorContainer) return;

		const theme = EditorView.theme({
			'&': {
				height: '100%',
				backgroundColor: 'transparent',
				fontSize: '0.875rem' // text-sm
			},
			'.cm-content': {
				fontFamily:
					"ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace",
				padding: '1rem'
			},
			'.cm-scroller': {
				fontFamily: 'inherit'
			},
			'&.cm-focused': {
				outline: 'none'
			}
		});

		const startState = EditorState.create({
			doc: sql,
			extensions: [
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
				theme,
				placeholder('SELECT * FROM table...'),
				EditorView.updateListener.of((update) => {
					if (update.docChanged) {
						sql = update.state.doc.toString();
					}
				})
			]
		});

		editorView = new EditorView({
			state: startState,
			parent: editorContainer
		});

		updateSchema();

		return () => {
			editorView?.destroy();
		};
	});
</script>

<div class="space-y-4">
	<div
		class="relative min-h-[150px] overflow-hidden rounded-md border bg-muted/50 font-mono text-sm"
	>
		<div bind:this={editorContainer} class="h-full w-full"></div>
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
