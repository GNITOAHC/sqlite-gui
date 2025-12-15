<script lang="ts">
	import { api } from '$lib/api/client';
	import * as Table from '$lib/components/ui/table/index.js';
	import ActionEllipsis from './action-ellipsis.svelte';
	import ActionInsert from './action-insert.svelte';
	import ActionDroptable from './action-droptable.svelte';

	let {
		table,
		db,
		onRefresh = undefined
	} = $props<{
		table: string;
		db: string;
		onRefresh?: () => void;
	}>();

	let tableCols = $state<any[] | null>(null);
	let tableRows = $state<any[] | null>(null);
	let error = $state<string | null>(null);
	let loading = $state(false);

	async function loadTableData() {
		loading = true;
		tableCols = null;
		tableRows = null;
		error = null;

		try {
			const [colsRes, rowsRes] = await Promise.all([
				api<any, { columns: any[] }>(`/tables/${table}/columns?db=${db}`),
				api<any, { rows: any[] }>(`/tables/${table}/rows?db=${db}`)
			]);

			tableCols = (colsRes.columns ?? []).map((col) => ({ ...col }));
			tableRows = (rowsRes.rows ?? []).map((row) => ({ ...row }));
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to load table data';
		} finally {
			loading = false;
		}
	}

	export function refresh() {
		loadTableData();
		onRefresh?.();
	}

	// Load data when table changes
	$effect(() => {
		if (table) {
			loadTableData();
		}
	});
</script>

<div class="space-y-4">
	<div class="flex flex-col rounded-lg border">
		<button onclick={() => console.log(tableCols)}>Console Cols</button>
		<button onclick={() => console.log(tableRows)}>Console Rows</button>
	</div>

	<div class="rounded-lg border p-4">
		<div class="flex items-center gap-3">
			<h3 class="text-lg font-semibold">Table: {table}</h3>
			<ActionDroptable {table} {db} />
		</div>
		{#key `${table}-${tableCols ? tableCols.length : 'none'}`}
			<ActionInsert cols={tableCols} {table} {db} onSuccess={refresh} />
		{/key}
		{#if loading}
			<p class="mt-2 text-sm text-muted-foreground">Loading table data…</p>
		{:else if error}
			<p class="mt-2 text-sm text-destructive">{error}</p>
		{:else if tableCols === null || tableRows === null}
			<p class="mt-2 text-sm text-muted-foreground">No data available.</p>
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
								{table}
								{db}
								onSuccess={refresh}
							/>
						</Table.Row>
					{/each}
				</Table.Body>
			</Table.Root>
		{/if}
	</div>
</div>
