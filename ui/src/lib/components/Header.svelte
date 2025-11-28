<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Database, Github } from '@lucide/svelte';
</script>

<header class="rounded-xl border bg-card shadow-sm">
	<div class="flex flex-col gap-5 p-3 sm:p-5">
		<div class="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
			<div class="flex items-center gap-3">
				<Database />
				<a class="text-lg leading-none font-semibold text-foreground" href="/"> SQLite Studio </a>
			</div>

			<Button variant="outline" size="sm" href="https://github.com/GNITOAHC/sqlite-gui">
				<Github />
			</Button>
		</div>

		<div class="flex flex-col gap-2 sm:hidden">
			<Button variant="outline" size="sm" class="w-full justify-center">
				<Github class="size-4" />
				GitHub
			</Button>
		</div>
	</div>
</header>
