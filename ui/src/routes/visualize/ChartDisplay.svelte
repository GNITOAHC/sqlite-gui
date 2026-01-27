<script lang="ts">
	import { onMount } from 'svelte';
	import { Chart, registerables } from 'chart.js';
	import { Label } from '$lib/components/ui/label/index.js';

	// Register all Chart.js components
	Chart.register(...registerables);

	type ChartType = 'line' | 'bar';
	type Row = Record<string, unknown>;

	interface Props {
		data: Row[];
		chartType: ChartType;
	}

	let { data, chartType }: Props = $props();

	let canvasRef = $state<HTMLCanvasElement | null>(null);
	let chartInstance: Chart | null = null;

	// Extract columns from data
	let columns = $derived(data.length > 0 ? Object.keys(data[0]) : []);

	// Find numeric columns
	let numericColumns = $derived(columns.filter((col) => data.some((row) => isNumeric(row[col]))));

	// Selected axis columns
	let xAxisColumn = $state<string>('');
	let yAxisColumn = $state<string>('');

	// Error and warning states
	let chartError = $state<string | null>(null);
	let chartWarning = $state<string | null>(null);
	let skippedRows = $state<number>(0);

	function isNumeric(value: unknown): boolean {
		if (value === null || value === undefined || value === '') {
			return false;
		}
		const num = Number(value);
		return !isNaN(num) && isFinite(num);
	}

	// Auto-select columns when data changes
	$effect(() => {
		if (columns.length > 0 && !xAxisColumn) {
			xAxisColumn = columns[0];
		}
		if (numericColumns.length > 0 && !yAxisColumn) {
			// Prefer the first numeric column that's not the x-axis
			const preferredY = numericColumns.find((col) => col !== xAxisColumn);
			yAxisColumn = preferredY ?? numericColumns[0];
		}
	});

	// Render chart when dependencies change
	$effect(() => {
		if (canvasRef && data.length > 0 && xAxisColumn && yAxisColumn) {
			renderChart();
		}
	});

	function renderChart() {
		chartError = null;
		chartWarning = null;
		skippedRows = 0;

		// Validate columns exist
		if (!columns.includes(xAxisColumn) || !columns.includes(yAxisColumn)) {
			chartError = 'Selected columns not found in data.';
			return;
		}

		// Check if Y column has any numeric values
		const hasNumericY = data.some((row) => isNumeric(row[yAxisColumn]));
		if (!hasNumericY) {
			chartError = `The selected Y-axis column "${yAxisColumn}" contains no numeric values. Please select a different column.`;
			return;
		}

		// Prepare chart data, filtering out non-numeric Y values
		const labels: string[] = [];
		const values: number[] = [];

		data.forEach((row) => {
			const yValue = row[yAxisColumn];
			if (isNumeric(yValue)) {
				labels.push(String(row[xAxisColumn] ?? ''));
				values.push(Number(yValue));
			} else {
				skippedRows++;
			}
		});

		if (values.length === 0) {
			chartError =
				'All values in the selected Y-axis column are non-numeric or null. Please select a different column or modify your query.';
			return;
		}

		if (skippedRows > 0) {
			chartWarning = `${skippedRows} row${skippedRows > 1 ? 's were' : ' was'} skipped because ${skippedRows > 1 ? 'they contain' : 'it contains'} non-numeric values in the Y-axis column.`;
		}

		// Destroy existing chart
		if (chartInstance) {
			chartInstance.destroy();
			chartInstance = null;
		}

		// Create new chart
		if (!canvasRef) {
			chartError = 'Canvas element not available.';
			return;
		}
		const ctx = canvasRef.getContext('2d');
		if (!ctx) {
			chartError = 'Failed to get canvas context.';
			return;
		}

		const chartColors = {
			line: {
				borderColor: 'oklch(0.646 0.222 41.116)',
				backgroundColor: 'oklch(0.646 0.222 41.116 / 0.1)',
				pointBackgroundColor: 'oklch(0.646 0.222 41.116)'
			},
			bar: {
				backgroundColor: 'oklch(0.6 0.118 184.704 / 0.8)',
				borderColor: 'oklch(0.6 0.118 184.704)',
				hoverBackgroundColor: 'oklch(0.6 0.118 184.704)'
			}
		};

		chartInstance = new Chart(ctx, {
			type: chartType,
			data: {
				labels,
				datasets: [
					{
						label: yAxisColumn,
						data: values,
						...(chartType === 'line'
							? {
									borderColor: chartColors.line.borderColor,
									backgroundColor: chartColors.line.backgroundColor,
									pointBackgroundColor: chartColors.line.pointBackgroundColor,
									fill: true,
									tension: 0.3
								}
							: {
									backgroundColor: chartColors.bar.backgroundColor,
									borderColor: chartColors.bar.borderColor,
									borderWidth: 1,
									hoverBackgroundColor: chartColors.bar.hoverBackgroundColor
								})
					}
				]
			},
			options: {
				responsive: true,
				maintainAspectRatio: false,
				plugins: {
					legend: {
						display: true,
						position: 'top'
					},
					tooltip: {
						enabled: true
					}
				},
				scales: {
					x: {
						title: {
							display: true,
							text: xAxisColumn
						}
					},
					y: {
						title: {
							display: true,
							text: yAxisColumn
						},
						beginAtZero: true
					}
				}
			}
		});
	}

	// Cleanup on unmount
	onMount(() => {
		return () => {
			if (chartInstance) {
				chartInstance.destroy();
			}
		};
	});
</script>

<div class="space-y-4">
	<!-- Axis Selectors -->
	<div class="flex flex-wrap gap-4">
		<div class="flex flex-col gap-2">
			<Label for="x-axis">X-axis Column</Label>
			<select
				id="x-axis"
				class="h-9 rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50"
				bind:value={xAxisColumn}
			>
				{#each columns as col}
					<option value={col}>{col}</option>
				{/each}
			</select>
		</div>

		<div class="flex flex-col gap-2">
			<Label for="y-axis">Y-axis Column (numeric)</Label>
			<select
				id="y-axis"
				class="h-9 rounded-md border border-input bg-background px-3 py-1 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50"
				bind:value={yAxisColumn}
			>
				{#each numericColumns as col}
					<option value={col}>{col}</option>
				{/each}
				{#if numericColumns.length === 0}
					<option value="" disabled>No numeric columns</option>
				{/if}
			</select>
		</div>
	</div>

	<!-- Error Display -->
	{#if chartError}
		<div
			class="rounded-md border border-destructive/20 bg-destructive/10 px-4 py-3 text-sm text-destructive"
		>
			<strong>Error:</strong>
			{chartError}
		</div>
	{/if}

	<!-- Warning Display -->
	{#if chartWarning}
		<div
			class="rounded-md border border-yellow-500/20 bg-yellow-500/10 px-4 py-3 text-sm text-yellow-700 dark:text-yellow-400"
		>
			<strong>Warning:</strong>
			{chartWarning}
		</div>
	{/if}

	<!-- Chart Canvas -->
	{#if !chartError}
		<div class="relative h-[400px] w-full rounded-md border bg-background p-4">
			<canvas bind:this={canvasRef}></canvas>
		</div>
	{/if}
</div>
