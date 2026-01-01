<script lang="ts">
	import { api } from '$lib/api/client';
	import * as Table from '$lib/components/ui/table/index.js';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { RefreshCcw, Columns2 } from '@lucide/svelte';
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
	let hiddenColumns = $state<Set<string>>(new Set());

	const visibleCols = $derived(tableCols?.filter((col) => !hiddenColumns.has(col.Name)) ?? []);

	function toggleColumn(colName: string) {
		if (hiddenColumns.has(colName)) {
			hiddenColumns.delete(colName);
		} else {
			hiddenColumns.add(colName);
		}
		hiddenColumns = new Set(hiddenColumns);
	}

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
	<div class="rounded-lg border p-4">
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-3">
				<h3 class="text-lg font-semibold">Table: {table}</h3>
				<ActionDroptable {table} {db} />
			</div>
			<div class="flex items-center gap-2">
				<Button variant="ghost" size="icon" onclick={() => refresh()} title="Refresh">
					<RefreshCcw class="h-4 w-4" />
				</Button>
				{#if tableCols && tableCols.length > 0}
					<DropdownMenu.Root>
						<DropdownMenu.Trigger>
							{#snippet child({ props })}
								<Button {...props} variant="ghost" size="icon" title="Toggle columns">
									<Columns2 class="h-4 w-4" />
								</Button>
							{/snippet}
						</DropdownMenu.Trigger>
						<DropdownMenu.Content align="end">
							<DropdownMenu.Group>
								<DropdownMenu.Label>Visible Columns</DropdownMenu.Label>
								<DropdownMenu.Separator />
								{#each tableCols as col}
									<DropdownMenu.CheckboxItem
										checked={!hiddenColumns.has(col.Name)}
										onCheckedChange={() => toggleColumn(col.Name)}
									>
										{col.Name}
									</DropdownMenu.CheckboxItem>
								{/each}
							</DropdownMenu.Group>
						</DropdownMenu.Content>
					</DropdownMenu.Root>
				{/if}
			</div>
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
						{#each visibleCols as col}
							<Table.Head>{col.Name}</Table.Head>
						{/each}
					</Table.Row>
				</Table.Header>
				<Table.Body>
					{#each tableRows as row}
						<Table.Row class="relative">
							{#each visibleCols as col}
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
