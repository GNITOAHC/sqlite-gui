<script lang="ts">
	import { api } from '$lib/api/client';
	import * as Table from '$lib/components/ui/table/index.js';
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { RefreshCcw, Copy, Check, ChevronDown, ChevronLeft, ChevronRight } from '@lucide/svelte';
	import ActionEllipsis from './action-ellipsis.svelte';
	import ActionInsert from './action-insert.svelte';
	import ActionDroptable from './action-droptable.svelte';
	import JSONViewer from '$lib/components/JSONViewer.svelte';

	let {
		table,
		db,
		onRefresh = undefined
	} = $props<{
		table: string;
		db: string;
		onRefresh?: () => void;
	}>();

	// Phase 1: column metadata
	let tableCols = $state<any[] | null>(null);
	let colsLoading = $state(false);
	let colsError = $state<string | null>(null);

	// Column selection (all selected by default after loadColumns)
	let selectedColumns = $state<Set<string>>(new Set());

	// Pagination config
	const PAGE_SIZE_OPTIONS = [10, 25, 50, 100, 500];
	let pageSize = $state(25);
	let currentPage = $state(0);

	// Phase 2: row data
	let tableRows = $state<any[] | null>(null);
	let totalRows = $state(0);
	let rowsLoading = $state(false);
	let rowsError = $state<string | null>(null);
	let dataLoaded = $state(false);

	// Cell dialog state
	let cellDialogOpen = $state(false);
	let selectedCellData = $state<any>(null);
	let selectedColName = $state<string>('');
	let copied = $state(false);
	let viewMode = $state<'text' | 'json'>('text');

	// Derived
	const totalPages = $derived(Math.ceil(totalRows / pageSize) || 1);
	const showingFrom = $derived(currentPage * pageSize + 1);
	const showingTo = $derived(Math.min((currentPage + 1) * pageSize, totalRows));
	// Empty array means "all columns" — backend will use SELECT *
	const selectedColList = $derived(
		selectedColumns.size > 0 && tableCols && selectedColumns.size < tableCols.length
			? [...selectedColumns]
			: []
	);
	const visibleCols = $derived(tableCols?.filter((c) => selectedColumns.has(c.Name)) ?? []);

	function openCellDialog(colName: string, cellData: any) {
		selectedColName = colName;
		selectedCellData = cellData;
		cellDialogOpen = true;
		copied = false;
		viewMode = 'text';
	}

	async function copyToClipboard() {
		await navigator.clipboard.writeText(String(selectedCellData ?? ''));
		copied = true;
		setTimeout(() => (copied = false), 2000);
	}

	function toggleColumn(colName: string) {
		if (selectedColumns.has(colName)) {
			selectedColumns.delete(colName);
		} else {
			selectedColumns.add(colName);
		}
		selectedColumns = new Set(selectedColumns);
	}

	// Phase 1: fetch only column metadata — called automatically on table change
	async function loadColumns() {
		colsLoading = true;
		tableCols = null;
		tableRows = null;
		totalRows = 0;
		dataLoaded = false;
		currentPage = 0;
		colsError = null;
		rowsError = null;
		try {
			const res = await api<any, { columns: any[] }>(`/tables/${table}/columns?db=${db}`);
			tableCols = (res.columns ?? []).map((col) => ({ ...col }));
			selectedColumns = new Set(tableCols.map((c) => c.Name));
		} catch (err) {
			colsError = err instanceof Error ? err.message : 'Failed to load columns';
		} finally {
			colsLoading = false;
		}
	}

	// Phase 2: fetch paginated rows — called only on explicit user action or page navigation
	async function loadRows() {
		if (!tableCols) return;
		rowsLoading = true;
		rowsError = null;
		try {
			const offset = currentPage * pageSize;
			const colParam =
				selectedColList.length > 0
					? `&columns=${encodeURIComponent(selectedColList.join(','))}`
					: '';
			const res = await api<any, { rows: any[]; total: number }>(
				`/tables/${table}/rows?db=${db}&limit=${pageSize}&offset=${offset}${colParam}`
			);
			tableRows = (res.rows ?? []).map((row) => ({ ...row }));
			totalRows = res.total ?? 0;
			dataLoaded = true;
		} catch (err) {
			rowsError = err instanceof Error ? err.message : 'Failed to load rows';
		} finally {
			rowsLoading = false;
		}
	}

	function handleLoadData() {
		currentPage = 0;
		loadRows();
	}

	function prevPage() {
		if (currentPage > 0) {
			currentPage--;
			loadRows();
		}
	}

	function nextPage() {
		if (currentPage < totalPages - 1) {
			currentPage++;
			loadRows();
		}
	}

	function changePageSize(size: number) {
		pageSize = size;
		currentPage = 0;
		if (dataLoaded) loadRows();
	}

	let pageInputValue = $state('');

	function syncPageInput() {
		pageInputValue = String(currentPage + 1);
	}

	function commitPageInput() {
		const num = parseInt(pageInputValue, 10);
		if (!isNaN(num) && num >= 1 && num <= totalPages && num - 1 !== currentPage) {
			currentPage = num - 1;
			loadRows();
		} else {
			syncPageInput();
		}
	}

	$effect(() => {
		syncPageInput();
	});

	export function refresh() {
		if (dataLoaded) loadRows();
		onRefresh?.();
	}

	// Only column metadata loads automatically; rows require explicit user action
	$effect(() => {
		if (table) {
			loadColumns();
		}
	});
</script>

<div class="space-y-4">
	<div class="rounded-lg border p-4">
		<!-- Header -->
		<div class="flex items-center justify-between">
			<div class="flex items-center gap-3">
				<h3 class="text-lg font-semibold">Table: {table}</h3>
				<ActionDroptable {table} {db} />
			</div>
			<Button variant="ghost" size="icon" onclick={() => refresh()} title="Refresh">
				<RefreshCcw class="h-4 w-4" />
			</Button>
		</div>

		{#key `${table}-${tableCols ? tableCols.length : 'none'}`}
			<ActionInsert cols={tableCols} {table} {db} onSuccess={refresh} />
		{/key}

		{#if colsLoading}
			<p class="mt-2 text-sm text-muted-foreground">Loading columns…</p>
		{:else if colsError}
			<p class="mt-2 text-sm text-destructive">{colsError}</p>
		{:else if tableCols !== null}
			<!-- Config bar: column selector + page size + load button -->
			<div class="mt-3 flex flex-wrap items-center gap-2">
				<!-- Column selector dropdown -->
				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<Button {...props} variant="outline" size="sm">
								{selectedColumns.size}/{tableCols?.length ?? 0} columns
								<ChevronDown class="ml-1 h-3 w-3" />
							</Button>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content align="start" class="max-h-64 overflow-y-auto">
						<DropdownMenu.Group>
							<DropdownMenu.Label>Select Columns</DropdownMenu.Label>
							<DropdownMenu.Separator />
							{#each tableCols as col}
								<DropdownMenu.CheckboxItem
									checked={selectedColumns.has(col.Name)}
									onCheckedChange={() => toggleColumn(col.Name)}
									closeOnSelect={false}
								>
									{col.Name}
									<span class="ml-1 text-xs text-muted-foreground">{col.Type}</span>
								</DropdownMenu.CheckboxItem>
							{/each}
						</DropdownMenu.Group>
					</DropdownMenu.Content>
				</DropdownMenu.Root>

				<!-- Page size selector -->
				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<Button {...props} variant="outline" size="sm">
								{pageSize} rows/page
								<ChevronDown class="ml-1 h-3 w-3" />
							</Button>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content align="start">
						{#each PAGE_SIZE_OPTIONS as size}
							<DropdownMenu.Item onSelect={() => changePageSize(size)}>
								{size}{size === pageSize ? ' ✓' : ''}
							</DropdownMenu.Item>
						{/each}
					</DropdownMenu.Content>
				</DropdownMenu.Root>

				<!-- Load Data button -->
				<Button
					size="sm"
					onclick={handleLoadData}
					disabled={rowsLoading || selectedColumns.size === 0}
				>
					{rowsLoading ? 'Loading…' : dataLoaded ? 'Reload' : 'Load Data'}
				</Button>
			</div>

			{#if rowsError}
				<p class="mt-2 text-sm text-destructive">{rowsError}</p>
			{/if}

			{#if dataLoaded && tableRows !== null}
				<!-- Data table -->
				<Table.Root class="mt-3">
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
									<Table.Cell
										class="max-w-[200px] cursor-pointer overflow-hidden text-ellipsis hover:bg-muted/70"
										onclick={() => openCellDialog(col.Name, row[col.Name])}
									>
										{row[col.Name]}
									</Table.Cell>
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

				<!-- Pagination -->
				<div class="mt-2 flex items-center justify-between text-sm text-muted-foreground">
					<span>
						{#if totalRows > 0}
							Showing {showingFrom}–{showingTo} of {totalRows}
						{:else}
							No rows
						{/if}
					</span>
					<div class="flex items-center gap-1">
						<Button
							variant="ghost"
							size="icon"
							onclick={prevPage}
							disabled={currentPage === 0 || rowsLoading}
							title="Previous page"
						>
							<ChevronLeft class="h-4 w-4" />
						</Button>
						<span class="flex items-center gap-1 px-1">
							<span class="text-muted-foreground">Page</span>
							<input
								class="rounded border bg-background px-1 py-0.5 text-center text-sm focus:ring-1 focus:ring-ring focus:outline-none"
								style="width: {Math.max(String(totalPages).length + 2, 4)}ch"
								type="text"
								inputmode="numeric"
								pattern="[0-9]*"
								value={pageInputValue}
								disabled={rowsLoading}
								oninput={(e) => (pageInputValue = (e.target as HTMLInputElement).value)}
								onblur={commitPageInput}
								onkeydown={(e) => {
									if (e.key === 'Enter') {
										(e.target as HTMLInputElement).blur();
									}
								}}
							/>
							<span class="text-muted-foreground">of {totalPages}</span>
						</span>
						<Button
							variant="ghost"
							size="icon"
							onclick={nextPage}
							disabled={currentPage >= totalPages - 1 || rowsLoading}
							title="Next page"
						>
							<ChevronRight class="h-4 w-4" />
						</Button>
					</div>
				</div>
			{:else if !rowsLoading}
				<p class="mt-3 text-sm text-muted-foreground">
					Configure columns and page size above, then click Load Data.
				</p>
			{/if}
		{/if}
	</div>

	<!-- Cell Data Dialog -->
	<Dialog.Root bind:open={cellDialogOpen}>
		<Dialog.Content class="sm:max-w-[600px]">
			<Dialog.Header>
				<Dialog.Title>{selectedColName}</Dialog.Title>
			</Dialog.Header>
			{#if viewMode === 'text'}
				<div
					class="max-h-[60vh] overflow-auto rounded border bg-muted/30 p-4 font-mono text-sm break-all whitespace-pre-wrap"
				>
					{selectedCellData}
				</div>
			{:else}
				<JSONViewer data={selectedCellData} defaultExpandLevel={1} />
			{/if}
			<Dialog.Footer class="flex items-center justify-between sm:justify-between">
				<DropdownMenu.Root>
					<DropdownMenu.Trigger>
						{#snippet child({ props })}
							<Button {...props} variant="outline" size="sm">
								{viewMode === 'text' ? 'Plain Text' : 'JSON'}
								<ChevronDown class="ml-2 h-4 w-4" />
							</Button>
						{/snippet}
					</DropdownMenu.Trigger>
					<DropdownMenu.Content align="start">
						<DropdownMenu.RadioGroup bind:value={viewMode}>
							<DropdownMenu.RadioItem value="text">Plain Text</DropdownMenu.RadioItem>
							<DropdownMenu.RadioItem value="json">JSON</DropdownMenu.RadioItem>
						</DropdownMenu.RadioGroup>
					</DropdownMenu.Content>
				</DropdownMenu.Root>
				<Button variant="outline" onclick={copyToClipboard}>
					{#if copied}
						<Check class="mr-2 h-4 w-4" />
						Copied!
					{:else}
						<Copy class="mr-2 h-4 w-4" />
						Copy
					{/if}
				</Button>
			</Dialog.Footer>
		</Dialog.Content>
	</Dialog.Root>
</div>
