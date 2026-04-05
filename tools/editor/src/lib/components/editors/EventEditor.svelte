<script lang="ts">
	import { projectStore } from '../../store/project.svelte.js';

	let { eventName }: { eventName: string } = $props();

	let event = $derived(projectStore.project.events.find((e) => e.data.event === eventName));
</script>

{#if event}
	<div class="space-y-6">
		<h2 class="text-xl font-bold text-gray-100">
			<span class="text-gray-500">Event:</span> {eventName}
		</h2>

		<!-- Conditions -->
		<div>
			<h3 class="mb-2 text-sm font-semibold text-gray-300">
				Conditions
				<span class="font-normal text-gray-600">({event.data.conditions.length})</span>
			</h3>
			<div class="rounded border border-gray-700 bg-gray-800/50 p-4 text-sm text-gray-500">
				{#if event.data.conditions.length > 0}
					{#each event.data.conditions as cond}
						<div class="py-0.5 font-mono text-xs text-gray-400">{cond.type}: {JSON.stringify(cond)}</div>
					{/each}
				{:else}
					No conditions — triggers on every event.
				{/if}
			</div>
		</div>

		<!-- Effects -->
		<div>
			<h3 class="mb-2 text-sm font-semibold text-gray-300">
				Effects
				<span class="font-normal text-gray-600">({event.data.effects.length})</span>
			</h3>
			<div class="rounded border border-gray-700 bg-gray-800/50 p-4 text-sm text-gray-500">
				{#if event.data.effects.length > 0}
					{#each event.data.effects as effect, i}
						<div class="py-0.5 font-mono text-xs text-gray-400">{i + 1}. {effect.type}: {JSON.stringify(effect)}</div>
					{/each}
				{:else}
					No effects defined.
				{/if}
			</div>
		</div>
	</div>
{/if}
