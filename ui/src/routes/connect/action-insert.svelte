<script lang="ts">
	import { onMount } from 'svelte';
	import { Button } from '$lib/components/ui/button/index.js';
	import { api } from '$lib/api/client';

	type Column = {
		Name: string;
		Type?: string;
		NotNull?: boolean;
		PrimaryKey?: boolean;
		Default?: unknown;
	};

	let { cols, table, db } = $props<{ cols: Column[] | null; table: string | null; db: string }>();

	let formValues = $state<Record<string, string>>({});
	let isSubmitting = $state(false);
	let message = $state<string | null>(null);

	let open = $state(false);

	function resetForm(nextCols: Column[] | null) {
		if (!nextCols) {
			formValues = {};
			return;
		}
		const fresh: Record<string, string> = {};
		for (const col of nextCols) {
			fresh[col.Name] = '';
		}
		formValues = fresh;
	}

	onMount(() => {
		resetForm(cols);
	});

	function updateField(name: string, value: string) {
		formValues = { ...formValues, [name]: value };
	}

	async function handleSubmit(event: Event) {
		event.preventDefault();
		if (!table || !cols) return;
		isSubmitting = true;
		message = null;

		try {
			const payload: Record<string, unknown> = {};
			for (const col of cols) {
				if (formValues[col.Name] === '') continue; // Ignore empty values
				payload[col.Name] = formValues[col.Name] ?? null;
			}

			await api(`/tables/${table}/rows?db=${db}`, {
				method: 'POST',
				body: payload
			});

			message = 'Inserted successfully.';
			resetForm(cols);
			if (typeof window !== 'undefined') {
				window.location.reload();
			}
		} catch (err) {
			message = err instanceof Error ? err.message : 'Insert failed';
		} finally {
			isSubmitting = false;
		}
	}
</script>

{#if !open}
	<Button class="w-full" variant="ghost" onclick={() => (open = true)}>Insert</Button>
{:else}
	<form class="mt-3 space-y-3 rounded-lg border p-4" onsubmit={handleSubmit}>
		<div class="flex items-center justify-between">
			<h4 class="font-semibold">Insert Row</h4>
			<div class="flex gap-2">
				<Button type="submit" size="sm" disabled={!table || !cols || isSubmitting}>
					{isSubmitting ? 'Inserting…' : 'Insert'}
				</Button>
				<Button size="sm" onclick={() => (open = false)}>Close</Button>
			</div>
		</div>

		{#if cols && cols.length > 0}
			<div class="grid gap-3 sm:grid-cols-2">
				{#each cols as col}
					<label class="flex flex-col gap-1 text-sm">
						<span class="font-medium text-foreground">{col.Name}</span>
						<input
							class="w-full rounded-md border bg-background px-3 py-2 text-sm transition outline-none focus:border-ring focus:ring-2 focus:ring-ring/50"
							type="text"
							value={formValues[col.Name] ?? ''}
							oninput={(e) => updateField(col.Name, (e.currentTarget as HTMLInputElement).value)}
							placeholder={col.Type || 'text'}
						/>
					</label>
				{/each}
			</div>
		{:else}
			<p class="text-sm text-muted-foreground">No columns to insert.</p>
		{/if}

		{#if message}
			<p class="text-sm text-muted-foreground">{message}</p>
		{/if}
	</form>
{/if}
