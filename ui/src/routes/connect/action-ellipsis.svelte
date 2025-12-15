<script lang="ts">
	import * as DropdownMenu from '$lib/components/ui/dropdown-menu/index.js';
	import { Ellipsis } from '@lucide/svelte';
	import { Button } from '$lib/components/ui/button/index.js';
	import { api } from '$lib/api/client';

	let { className, row, cols, table, db, onSuccess } = $props<{
		className?: string;
		row: any;
		cols: any[];
		table: string;
		db: string;
		onSuccess?: () => void;
	}>();

	const pks = () => {
		let keys = [];
		let vals = [];
		for (const c of cols) {
			if (c.PrimaryKey) {
				keys.push(c.Name);
				vals.push(row[c.Name]);
			}
		}
		return { keys, vals };
	};

	const deleteRow = async () => {
		// api<>("/tables/{table}/rows/{id}")
		const { keys, vals } = pks();

		console.log(keys, vals);
		console.log(row);
		console.log(cols);

		const resp = await api<any, any>(
			`/tables/${table}/rows/${vals.join(',')}?pk=${keys.join(',')}&db=${db}`,
			{ method: 'DELETE' }
		);

		console.log(JSON.stringify(resp));
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
				<DropdownMenu.Item>Edit</DropdownMenu.Item>
				<DropdownMenu.Item class="text-destructive" onclick={() => deleteRow()}
					>Delete</DropdownMenu.Item
				>
			</DropdownMenu.Group>
		</DropdownMenu.Content>
	</DropdownMenu.Root>
</div>
