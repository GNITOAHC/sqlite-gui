<script lang="ts">
	import { replaceState } from '$app/navigation';
	import { onMount } from 'svelte';
	import { api } from '$lib/api/client';
	import { Button } from '$lib/components/ui/button/index.js';
	import ActionNewtable from './action-newtable.svelte';
	import ActionSql from './action-sql.svelte';
	import TableView from './TableView.svelte';

	let db = '';
	let initialTable: string | null = null;

	// Get all tables
	let tables: string[] | null = null;
	let selectedTable: string | null = null;
	let error: string | null = null;
	let tableViewRef: TableView;

	async function loadTables() {
		try {
			const res = await api<any, { tables: string[] }>(`/tables?db=${db}`);
			tables = res.tables ?? [];
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load tables';
		}
	}

	function handleSelect(table: string) {
		selectedTable = table;

		const params = new URLSearchParams(window.location.search);
		params.set('table', table);
		replaceState(`${window.location.pathname}?${params}`, {});
	}

	onMount(() => {
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
		<TableView bind:this={tableViewRef} table={selectedTable} {db} />
	{:else}
		<p class="text-sm text-muted-foreground">Select a table to view its data.</p>
	{/if}
</div>
