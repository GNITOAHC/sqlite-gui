<script lang="ts">
	import { Plus, Trash } from '@lucide/svelte';
	import { Button, buttonVariants } from '$lib/components/ui/button/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Label } from '$lib/components/ui/label/index.js';
	import { api } from '$lib/api/client';

	type ColumnInput = {
		name: string;
		type: string;
		primaryKey: boolean;
		notNull: boolean;
		default?: string;
	};

	let { db } = $props<{ db: string }>();

	let open = $state(false);
	let tableName = $state('');
	let columns: ColumnInput[] = $state([
		{ name: 'id', type: 'INTEGER', primaryKey: true, notNull: true, default: undefined }
	]);
	let isSubmitting = $state(false);
	let message: string | null = $state(null);
	let error: string | null = $state(null);

	function addColumn() {
		columns = [
			...columns,
			{
				name: `col_${columns.length + 1}`,
				type: 'TEXT',
				primaryKey: false,
				notNull: false,
				default: undefined
			}
		];
	}

	function removeColumn(index: number) {
		columns = columns.filter((_, i) => i !== index);
	}

	function updateColumn<T extends keyof ColumnInput>(index: number, key: T, value: ColumnInput[T]) {
		columns = columns.map((col, i) => (i === index ? { ...col, [key]: value } : col));
	}

	const canSubmit = () => {
		const hasPrimary = columns.some((c) => c.primaryKey);
		return (
			tableName.trim().length > 0 &&
			columns.length > 0 &&
			hasPrimary &&
			columns.every((c) => c.name.trim().length > 0 && c.type.trim().length > 0)
		);
	};

	async function handleSubmit(event: Event) {
		event.preventDefault();
		if (!canSubmit()) {
			error = 'Provide a name, valid columns, and at least one primary key.';
			return;
		}
		isSubmitting = true;
		message = null;
		error = null;

		try {
			await api(`/tables?db=${db}`, {
				method: 'POST',
				body: {
					name: tableName.trim(),
					columns: columns.map((c) => ({
						name: c.name.trim(),
						type: c.type.trim(),
						primaryKey: c.primaryKey || undefined,
						notNull: c.notNull || undefined,
						default: c.default?.length ? c.default : undefined
					}))
				}
			});

			message = 'Table created. Reloading…';
			open = false;
			tableName = '';
			columns = [
				{ name: 'id', type: 'INTEGER', primaryKey: true, notNull: true, default: undefined }
			];

			if (typeof window !== 'undefined') {
				window.location.reload();
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Failed to create table';
		} finally {
			isSubmitting = false;
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Trigger
		class={buttonVariants({ variant: 'outline', size: 'icon' })}
		aria-label="New table"
	>
		<Plus />
	</Dialog.Trigger>
	<Dialog.Content class="sm:max-w-[520px]">
		<Dialog.Header>
			<Dialog.Title>Create Table</Dialog.Title>
		</Dialog.Header>

		<form class="space-y-4" onsubmit={handleSubmit}>
			<div class="grid gap-3">
				<div class="grid grid-cols-4 items-center gap-3">
					<Label class="text-end">Name</Label>
					<Input
						class="col-span-3"
						placeholder="memberships"
						value={tableName}
						oninput={(e) => (tableName = (e.currentTarget as HTMLInputElement).value)}
						required
					/>
				</div>
			</div>

			<div class="space-y-2">
				<div class="flex items-center justify-between">
					<Label>Columns</Label>
					<Button type="button" size="sm" variant="outline" onclick={addColumn}>
						<Plus class="size-4" /> Add column
					</Button>
				</div>

				<div class="space-y-3 max-h-80 overflow-y-auto pr-2">
					{#each columns as col, index}
						<div class="rounded-lg border p-3">
							<div class="grid gap-3 sm:grid-cols-3">
								<label class="flex flex-col gap-1 text-sm">
									<span class="font-medium">Name</span>
									<Input
										placeholder="user_id"
										value={col.name}
										oninput={(e) =>
											updateColumn(index, 'name', (e.currentTarget as HTMLInputElement).value)}
										required
									/>
								</label>
								<label class="flex flex-col gap-1 text-sm">
									<span class="font-medium">Type</span>
									<Input
										placeholder="INTEGER"
										value={col.type}
										oninput={(e) =>
											updateColumn(index, 'type', (e.currentTarget as HTMLInputElement).value)}
										required
									/>
								</label>
								<label class="flex flex-col gap-1 text-sm">
									<span class="font-medium">Default</span>
									<Input
										placeholder="member"
										value={col.default ?? ''}
										oninput={(e) =>
											updateColumn(index, 'default', (e.currentTarget as HTMLInputElement).value)}
									/>
								</label>
							</div>
							<div class="mt-3 flex flex-wrap items-center gap-4 text-sm">
								<label class="inline-flex items-center gap-2">
									<input
										type="checkbox"
										checked={col.primaryKey}
										onchange={(e) =>
											updateColumn(
												index,
												'primaryKey',
												(e.currentTarget as HTMLInputElement).checked
											)}
									/>
									<span>Primary key</span>
								</label>
								<label class="inline-flex items-center gap-2">
									<input
										type="checkbox"
										checked={col.notNull}
										onchange={(e) =>
											updateColumn(index, 'notNull', (e.currentTarget as HTMLInputElement).checked)}
									/>
									<span>Not null</span>
								</label>
								{#if columns.length > 1}
									<Button
										type="button"
										variant="ghost"
										size="sm"
										onclick={() => removeColumn(index)}
										class="text-destructive hover:text-destructive"
									>
										<Trash class="size-4" />
										Remove
									</Button>
								{/if}
							</div>
						</div>
					{/each}
				</div>
			</div>

			{#if error}
				<p class="text-sm text-destructive">{error}</p>
			{:else if message}
				<p class="text-sm text-muted-foreground">{message}</p>
			{:else if !columns.some((c) => c.primaryKey)}
				<p class="text-sm text-destructive">Mark at least one column as primary key.</p>
			{/if}

			<Dialog.Footer class="flex items-center justify-end gap-2">
				<Dialog.Close>
					<Button variant="ghost" type="button">Cancel</Button>
				</Dialog.Close>
				<Button type="submit" disabled={isSubmitting || !canSubmit()}>
					{isSubmitting ? 'Creating…' : 'Create'}
				</Button>
			</Dialog.Footer>
		</form>
	</Dialog.Content>
</Dialog.Root>
