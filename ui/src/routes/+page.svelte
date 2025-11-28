<script lang="ts">
	import { api } from '$lib/api/client';
	import { onMount } from 'svelte';
	import * as Table from '$lib/components/ui/table/index.js';
	import { Button } from '$lib/components/ui/button/index.js';

	let connections: any = null;
	let tables: any = null;
	onMount(async () => {
		connections = await api('/connections');
		connections = connections.connections;
		tables = await api('/tables');
		console.log(tables);
	});
</script>

<div>
	{#if connections}
		<h2 class="mt-8 mb-4">Available Connections</h2>
		<Table.Root>
			<Table.Header>
				<Table.Row>
					<Table.Head class="w-[100px]">Name</Table.Head>
					<Table.Head>ConnString</Table.Head>
					<!-- <Table.Head class="text-end">Connect</Table.Head> -->
				</Table.Row>
			</Table.Header>
			<Table.Body>
				{#each connections as connection}
					<Table.Row>
						<Table.Cell class="font-medium">{connection.name}</Table.Cell>
						<Table.Cell>{connection.connString}</Table.Cell>
						<Table.Cell class="text-end"
							><Button
								onclick={() => (window.location.href = `/connect?db=${connection.name}`)}
								variant="ghost">Connect</Button
							></Table.Cell
						>
					</Table.Row>
				{/each}
			</Table.Body>
		</Table.Root>
	{:else}
		<p>Loading connections...</p>
	{/if}
</div>
