<script lang="ts">
	import { projectStore } from '../store/project.svelte.js';

	let { onimport, onexport }: { onimport: () => void; onexport: () => void } = $props();

	let title = $derived(projectStore.project.meta.data.title || 'Untitled Project');
</script>

<header class="flex h-12 items-center justify-between border-b border-gray-800 bg-gray-900 px-4">
	<div class="flex items-center gap-3">
		<span class="text-sm font-bold tracking-tight text-gray-100">QuestCore</span>
		<span class="text-sm text-gray-500">/</span>
		<span class="text-sm text-gray-300">{title}</span>
	</div>

	<div class="flex items-center gap-2">
		<button
			onclick={onimport}
			class="rounded px-3 py-1.5 text-xs font-medium text-gray-300 hover:bg-gray-800 hover:text-gray-100"
		>
			Import
		</button>
		<button
			onclick={onexport}
			class="rounded bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-500"
		>
			Export
		</button>
	</div>
</header>
