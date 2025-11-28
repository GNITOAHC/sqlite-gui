<script lang="ts">
	import { page } from '$app/state';
	import { replaceState } from '$app/navigation';
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import * as Table from '$lib/components/ui/table/index.js';
	import ActionEllipsis from './action-ellipsis.svelte';
	import ActionInsert from './action-insert.svelte';

	// Access a specific query parameter
	const paramValue = page.url.searchParams.get('db') ?? '';
	const initialTable = page.url.searchParams.get('table');

	// Get all tables
	let tables: string[] | null = null;
	let selectedTable: string | null = null;
	let tableCols: any[] | null = null;
	let tableRows: any[] | null = null;
	let error: string | null = null;
	let selectionToken = 0;

	async function loadTables() {
		try {
			const res = await api<any, { tables: string[] }>(`/tables?db=${paramValue}`);
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
				api<any, { columns: any[] }>(`/tables/${table}/columns?db=${paramValue}`),
				api<any, { rows: any[] }>(`/tables/${table}/rows?db=${paramValue}`)
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
		const params = new URLSearchParams(page.url.searchParams);
		params.set('table', table);
		replaceState(`${page.url.pathname}?${params.toString()}`, page.state);
		loadTableData(table, token);
	}

	onMount(() => {
		loadTables().then(() => {
			if (initialTable) {
				handleSelect(initialTable);
			}
		});
	});
</script>

<div class="space-y-4">
	<div class="rounded-lg border p-4">
		<p class="text-sm text-muted-foreground">Database: {paramValue || 'Unknown'}</p>

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
						<button
							class={`rounded-md border px-3 py-1 text-sm transition ${
								selectedTable === table
									? 'bg-primary text-primary-foreground'
									: 'hover:bg-accent hover:text-accent-foreground'
							}`}
							onclick={() => handleSelect(table)}
						>
							{table}
						</button>
					</li>
				{/each}
			</ul>
		{/if}
	</div>

	{#if selectedTable}
		<div class="flex flex-col rounded-lg border">
			<button onclick={() => console.log(tableCols)}>Console Cols</button>
			<button onclick={() => console.log(tableRows)}>Console Rows</button>
			<!-- <button onclick={() => makeColumns(tableCols)}>Make Columns</button> -->
		</div>

		<div class="rounded-lg border p-4">
			<h3 class="text-lg font-semibold">Table: {selectedTable}</h3>
			{#key `${selectedTable ?? 'none'}-${tableCols ? tableCols.length : 'none'}`}
				<ActionInsert cols={tableCols} table={selectedTable} db={paramValue} />
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
							<Table.Row>
								{#each tableCols as col}
									<Table.Cell>{row[col.Name]}</Table.Cell>
								{/each}
								<ActionEllipsis
									className="absolute right-2"
									{row}
									cols={tableCols}
									table={selectedTable}
									db={paramValue}
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
