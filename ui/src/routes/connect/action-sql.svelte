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

		const isSelect = /^\s*(SELECT|PRAGMA|WITH|VALUES|EXPLAIN)\b/i.test(sql.trim());
		isExec = !isSelect;

		try {
			result = await api(`/` + (isSelect ? 'query' : 'exec') + `?db=${db}`, {
				method: 'POST',
				body: { query: sql, args: [] }
			});
		} catch (e: any) {
			error = e.message;
		} finally {
			loading = false;
		}
	}

	async function updateSchema() {
		if (!db || !editorView) return;
		try {
			const { tables = [] } = await api<any, { tables: string[] }>(`/tables?db=${db}`);
			const schema: Record<string, string[]> = {};

			await Promise.all(
				tables.map(async (table) => {
					try {
						const { columns = [] } = await api<any, { columns: any[] }>(
							`/tables/${table}/columns?db=${db}`
						);
						schema[table] = columns.map((c) => c.Name);
					} catch {}
				})
			);

			editorView.dispatch({ effects: schemaConfig.reconfigure(sqlLang({ schema })) });
		} catch {}
	}

	$effect(() => {
		if (editorView && db) updateSchema();
	});

	onMount(() => {
		if (!editorContainer) return;

		// prettier-ignore
		const extensions = [
			keymap.of([{ key: 'Mod-Enter', run: () => { runQuery(); return true; } }, ...defaultKeymap, ...historyKeymap, ...completionKeymap, ...closeBracketsKeymap]),
			history(), drawSelection(), bracketMatching(), closeBrackets(), autocompletion(),
			syntaxHighlighting(defaultHighlightStyle, { fallback: true }),
			schemaConfig.of(sqlLang()), placeholder('SELECT * FROM table...'),
			EditorView.updateListener.of((u) => { if (u.docChanged) sql = u.state.doc.toString(); }),
			EditorView.theme({
				'&': { height: '100%', backgroundColor: 'transparent', fontSize: '0.875rem' },
				'.cm-content': { fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace", padding: '1rem' },
				'.cm-scroller': { fontFamily: 'inherit' },
				'&.cm-focused': { outline: 'none' }
			})
		];

		editorView = new EditorView({
			state: EditorState.create({ doc: sql, extensions }),
			parent: editorContainer
		});
		updateSchema();

		return () => editorView?.destroy();
	});

	let columns = $derived(result?.rows?.length ? Object.keys(result.rows[0]) : []);
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
		<div class="rounded-md bg-destructive/10 p-4 text-destructive">Error: {error}</div>
	{/if}

	{#if result}
		{#if !isExec}
			{#if columns.length > 0}
				<div class="overflow-x-auto rounded-md border">
					<Table.Root>
						<Table.Header>
							<Table.Row>
								{#each columns as col}
									<Table.Head>{col}</Table.Head>
								{/each}
							</Table.Row>
						</Table.Header>
						<Table.Body>
							{#each result.rows as row}
								<Table.Row>
									{#each columns as col}
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
			<div class="rounded-md border p-4">
				<p>Rows Affected: {result.rowsAffected}</p>
				<p>Last Insert ID: {result.lastInsertId}</p>
			</div>
		{/if}
	{/if}
</div>
