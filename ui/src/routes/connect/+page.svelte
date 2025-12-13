<script lang="ts">
	import { replaceState } from '$app/navigation';
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import * as Table from '$lib/components/ui/table/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import ActionEllipsis from './action-ellipsis.svelte';
	import ActionInsert from './action-insert.svelte';
	import ActionNewtable from './action-newtable.svelte';
	import ActionDroptable from './action-droptable.svelte';
	import ActionSql from './action-sql.svelte';

	let db = '';
	let initialTable: string | null = null;

	// Get all tables
	let tables: string[] | null = null;
	let selectedTable: string | null = null;
	let tableCols: any[] | null = null;
	let tableRows: any[] | null = null;
	let error: string | null = null;
	let selectionToken = 0;

	async function loadTables() {
		try {
			const res = await api<any, { tables: string[] }>(`/tables?db=${db}`);
			tables = res.tables ?? [];
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load tables';
		}
	}

	async function loadTableData(table: string, token: number) {
		tableCols = null;
		tableRows = null;

		try {
			const [colsRes, rowsRes] = await Promise.all([
				api<any, { columns: any[] }>(`/tables/${table}/columns?db=${db}`),
				api<any, { rows: any[] }>(`/tables/${table}/rows?db=${db}`)
			]);

			// Avoid setting state if selection changed mid-flight
			if (token !== selectionToken) return;

			tableCols = (colsRes.columns ?? []).map((col) => ({ ...col }));
			tableRows = (rowsRes.rows ?? []).map((row) => ({ ...row }));
		} catch (err) {
			if (token !== selectionToken) return;
			error = err instanceof Error ? err.message : 'Failed to load table data';
		}
	}

	function handleSelect(table: string) {
		selectedTable = table;
		selectionToken += 1;
		const token = selectionToken;

		const params = new URLSearchParams(window.location.search);
		params.set('table', table);
		replaceState(`${window.location.pathname}?${params}`, {});

		loadTableData(table, token);
	}

	onMount(() => {
		// const p = get(page).url.searchParams;
		const p = new URLSearchParams(window.location.search);
		db = p.get('db') ?? '';
		initialTable = p.get('table');

		loadTables().then(() => {
			if (initialTable) {
				handleSelect(initialTable);
			}
		});
	});
</script>

<div class="space-y-4">
	<div class="rounded-lg border p-4">
		<p class="text-sm text-muted-foreground">Database: {db || 'Unknown'}</p>

		{#if error}
			<p class="mt-2 text-sm text-destructive">{error}</p>
		{:else if tables === null}
			<p class="mt-2 text-sm text-muted-foreground">Loading tables...</p>
		{:else if tables.length === 0}
			<p class="mt-2 text-sm text-muted-foreground">No tables found.</p>
		{:else}
			<h2 class="mt-3 text-lg font-semibold">Tables</h2>
			<ul class="mt-2 flex flex-wrap gap-2">
				{#each tables as table}
					<li>
						<Button
							variant={selectedTable === table ? 'default' : 'outline'}
							onclick={() => handleSelect(table)}
						>
							{table}
						</Button>
					</li>
				{/each}
				<li>
					<ActionNewtable {db} />
				</li>
			</ul>
		{/if}
	</div>

	<ActionSql {db} />

	{#if selectedTable}
		<div class="flex flex-col rounded-lg border">
			<button onclick={() => console.log(tableCols)}>Console Cols</button>
			<button onclick={() => console.log(tableRows)}>Console Rows</button>
			<!-- <button onclick={() => makeColumns(tableCols)}>Make Columns</button> -->
		</div>

		<div class="rounded-lg border p-4">
			<div class="flex items-center gap-3">
				<h3 class="text-lg font-semibold">Table: {selectedTable}</h3>
				<ActionDroptable table={selectedTable} {db} />
			</div>
			{#key `${selectedTable ?? 'none'}-${tableCols ? tableCols.length : 'none'}`}
				<ActionInsert cols={tableCols} table={selectedTable} {db} />
			{/key}
			{#if tableCols === null || tableRows === null}
				<p class="mt-2 text-sm text-muted-foreground">Loading table data…</p>
			{:else}
				<Table.Root>
					<Table.Header>
						<Table.Row>
							{#each tableCols as col}
								<Table.Head>{col.Name}</Table.Head>
							{/each}
						</Table.Row>
					</Table.Header>
					<Table.Body>
						{#each tableRows as row}
							<Table.Row class="relative">
								{#each tableCols as col}
									<Table.Cell>{row[col.Name]}</Table.Cell>
								{/each}
								<ActionEllipsis
									className="sticky right-2 max-w-9"
									{row}
									cols={tableCols}
									table={selectedTable}
									{db}
								/>
							</Table.Row>
						{/each}
					</Table.Body>
				</Table.Root>
			{/if}
		</div>
	{:else}
		<p class="text-sm text-muted-foreground">Select a table to view its data.</p>
	{/if}
</div>
