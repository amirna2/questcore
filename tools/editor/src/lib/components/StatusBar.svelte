<script lang="ts">
	import { projectStore } from '../store/project.svelte.js';

	let roomCount = $derived(projectStore.project.rooms.length);
	let entityCount = $derived(projectStore.project.entities.length);
	let ruleCount = $derived(projectStore.project.rules.length);
	let eventCount = $derived(projectStore.project.events.length);
</script>

<footer class="flex h-7 items-center justify-between border-t border-gray-800 bg-gray-900 px-4 text-xs text-gray-500">
	<div class="flex gap-4">
		<span>{roomCount} {roomCount === 1 ? 'room' : 'rooms'}</span>
		<span>{entityCount} {entityCount === 1 ? 'entity' : 'entities'}</span>
		<span>{ruleCount} {ruleCount === 1 ? 'rule' : 'rules'}</span>
		<span>{eventCount} {eventCount === 1 ? 'event' : 'events'}</span>
	</div>
</footer>
