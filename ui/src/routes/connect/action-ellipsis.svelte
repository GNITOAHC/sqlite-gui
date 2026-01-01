<script lang="ts">
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Ellipsis } from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { api } from '$lib/api/client';

	let { className, row, cols, table, db, onSuccess } = $props<{
		className?: string;
		row: any;
		cols: any[];
		table: string;
		db: string;
		onSuccess?: () => void;
	}>();

	let editOpen = $state(false);
	let editValues = $state<Record<string, any>>({});
	let isSubmitting = $state(false);
	let editError = $state<string | null>(null);

	const pks = () => {
		let keys: string[] = [];
		let vals: any[] = [];
		for (const c of cols) {
			if (c.PrimaryKey) {
				keys.push(c.Name);
				vals.push(row[c.Name]);
			}
		}
		return { keys, vals };
	};

	function openEditDialog() {
		editValues = { ...row };
		editError = null;
		editOpen = true;
	}

	const editRow = async (e: Event) => {
		e.preventDefault();
		isSubmitting = true;
		editError = null;

		try {
			const { keys, vals } = pks();
			const payload: Record<string, any> = {};
			for (const c of cols) {
				if (!c.PrimaryKey) {
					payload[c.Name] = editValues[c.Name];
				}
			}

			await api<any, any>(`/tables/${table}/rows/${vals.join(',')}?pk=${keys.join(',')}&db=${db}`, {
				method: 'PUT',
				body: payload
			});

			editOpen = false;
			onSuccess?.();
		} catch (err) {
			editError = err instanceof Error ? err.message : 'Failed to update row';
		} finally {
			isSubmitting = false;
		}
	};

	const deleteRow = async () => {
		const { keys, vals } = pks();

		await api<any, any>(`/tables/${table}/rows/${vals.join(',')}?pk=${keys.join(',')}&db=${db}`, {
			method: 'DELETE'
		});

		onSuccess?.();
	};
</script>

<div class={className}>
	<DropdownMenu.Root>
		<DropdownMenu.Trigger>
			{#snippet child({ props })}
				<Button {...props} variant="ghost" size="icon"><Ellipsis /></Button>
			{/snippet}
		</DropdownMenu.Trigger>
		<DropdownMenu.Content>
			<DropdownMenu.Group>
				<DropdownMenu.Item onclick={() => openEditDialog()}>Edit</DropdownMenu.Item>
				<DropdownMenu.Item class="text-destructive" onclick={() => deleteRow()}
					>Delete</DropdownMenu.Item
				>
			</DropdownMenu.Group>
		</DropdownMenu.Content>
	</DropdownMenu.Root>

	<Dialog.Root bind:open={editOpen}>
		<Dialog.Content class="sm:max-w-[520px]">
			<Dialog.Header>
				<Dialog.Title>Edit Row</Dialog.Title>
				<Dialog.Description
					>Make changes to the row. Click save when you're done.</Dialog.Description
				>
			</Dialog.Header>
			<form onsubmit={editRow}>
				<div class="grid max-h-[60vh] gap-4 overflow-y-auto py-4">
					{#each cols as col}
						<div class="grid grid-cols-4 items-center gap-4">
							<Label class="text-right" for={col.Name}>
								{col.Name}
								{#if col.PrimaryKey}
									<span class="text-xs text-muted-foreground">(PK)</span>
								{/if}
							</Label>
							<Input
								id={col.Name}
								class="col-span-3"
								value={editValues[col.Name] ?? ''}
								oninput={(e) => (editValues[col.Name] = e.currentTarget.value)}
								disabled={col.PrimaryKey}
								placeholder={col.Type}
							/>
						</div>
					{/each}
				</div>
				{#if editError}
					<p class="mb-4 text-sm text-destructive">{editError}</p>
				{/if}
				<Dialog.Footer>
					<Button type="button" variant="ghost" onclick={() => (editOpen = false)}>Cancel</Button>
					<Button type="submit" disabled={isSubmitting}>
						{isSubmitting ? 'Saving…' : 'Save'}
					</Button>
				</Dialog.Footer>
			</form>
		</Dialog.Content>
	</Dialog.Root>
</div>
