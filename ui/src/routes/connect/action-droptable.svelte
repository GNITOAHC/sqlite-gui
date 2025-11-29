<script lang="ts">
	import { Trash } from '@lucide/svelte';
	import { Button, buttonVariants } from '$lib/components/ui/button/index.js';
	import * as Dialog from '$lib/components/ui/dialog/index.js';
	import { api } from '$lib/api/client';
	import { replaceState } from '$app/navigation';

	let open = $state(false);
	let { db, table } = $props<{ db: string; table: string }>();

	async function confirmDrop() {
		try {
			const resp = await api<any, { status: string }>(`/tables/${table}?db=${db}`, {
				method: 'DELETE'
			});
			if (resp.status === 'ok') {
				const params = new URLSearchParams(window.location.search);
				params.delete('table');
				replaceState(`${window.location.pathname}?${params}`, {});

				open = false;
				if (typeof window !== 'undefined') {
					window.location.reload();
				}
			}
		} catch (err) {
			console.error('Error dropping table:', err);
		}
	}
</script>

<Dialog.Root bind:open>
	<Dialog.Trigger
		class={buttonVariants({ variant: 'ghost', size: 'icon-sm' })}
		aria-label="Drop table"
	>
		<Trash />
	</Dialog.Trigger>
	<Dialog.Content class="sm:max-w-[520px]" showCloseButton={false}>
		<Dialog.Header>
			<Dialog.Title>Are you sure you want to drop {table}?</Dialog.Title>
		</Dialog.Header>
		<div class="grid grid-cols-2 gap-2">
			<Button variant="outline" onclick={() => confirmDrop()}>Drop</Button>
			<Dialog.Close>
				<Button variant="default" class="w-full">Cancel</Button>
			</Dialog.Close>
		</div>
	</Dialog.Content>
</Dialog.Root>
