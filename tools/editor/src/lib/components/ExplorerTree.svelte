<script lang="ts">
	import { projectStore, type Selection } from '../store/project.svelte.js';

	let collapsed = $state<Record<string, boolean>>({
		rooms: false,
		entities: false,
		rules: false,
		events: false
	});

	function toggle(section: string) {
		collapsed[section] = !collapsed[section];
	}

	function isSelected(kind: Selection['kind'], id: string | null): boolean {
		const sel = projectStore.selection;
		if (!sel) return false;
		return sel.kind === kind && sel.id === id;
	}

	function select(kind: Selection['kind'], id: string | null) {
		projectStore.select({ kind, id });
	}

	function entityKindLabel(kind: string): string {
		switch (kind) {
			case 'item': return 'item';
			case 'npc': return 'npc';
			case 'enemy': return 'enemy';
			default: return 'entity';
		}
	}

	let { onadd }: { onadd: (kind: string) => void } = $props();
</script>

<nav class="flex h-full flex-col overflow-y-auto bg-gray-900 text-sm">
	<div class="flex-1 py-2">
		<!-- Game Settings -->
		<button
			class="flex w-full items-center gap-2 px-3 py-1 text-left hover:bg-gray-800 {isSelected('game', null) ? 'bg-gray-800 text-white' : 'text-gray-300'}"
			onclick={() => select('game', null)}
		>
			<span class="text-gray-500">&#9881;</span>
			<span>Game Settings</span>
		</button>

		<!-- Rooms -->
		<div class="mt-2">
			<button
				class="flex w-full items-center gap-1 px-3 py-1 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 hover:text-gray-300"
				onclick={() => toggle('rooms')}
			>
				<span class="w-3 text-center">{collapsed.rooms ? '▸' : '▾'}</span>
				<span>Rooms</span>
				<span class="ml-auto text-gray-600">{projectStore.project.rooms.length}</span>
			</button>

			{#if !collapsed.rooms}
				{#each projectStore.project.rooms as room}
					<button
						class="flex w-full items-center gap-2 py-0.5 pl-7 pr-3 text-left hover:bg-gray-800 {isSelected('room', room.data.id) ? 'bg-gray-800 text-white' : 'text-gray-400'}"
						onclick={() => select('room', room.data.id)}
					>
						{room.data.id}
					</button>
				{/each}
			{/if}
		</div>

		<!-- Entities -->
		<div class="mt-2">
			<button
				class="flex w-full items-center gap-1 px-3 py-1 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 hover:text-gray-300"
				onclick={() => toggle('entities')}
			>
				<span class="w-3 text-center">{collapsed.entities ? '▸' : '▾'}</span>
				<span>Entities</span>
				<span class="ml-auto text-gray-600">{projectStore.project.entities.length}</span>
			</button>

			{#if !collapsed.entities}
				{#each projectStore.project.entities as entity}
					<button
						class="flex w-full items-center justify-between gap-2 py-0.5 pl-7 pr-3 text-left hover:bg-gray-800 {isSelected('entity', entity.data.id) ? 'bg-gray-800 text-white' : 'text-gray-400'}"
						onclick={() => select('entity', entity.data.id)}
					>
						<span class="truncate">{entity.data.name || entity.data.id}</span>
						<span class="shrink-0 text-xs text-gray-600">{entityKindLabel(entity.data.kind)}</span>
					</button>
				{/each}
			{/if}
		</div>

		<!-- Rules -->
		<div class="mt-2">
			<button
				class="flex w-full items-center gap-1 px-3 py-1 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 hover:text-gray-300"
				onclick={() => toggle('rules')}
			>
				<span class="w-3 text-center">{collapsed.rules ? '▸' : '▾'}</span>
				<span>Rules</span>
				<span class="ml-auto text-gray-600">{projectStore.project.rules.length}</span>
			</button>

			{#if !collapsed.rules}
				{#each projectStore.project.rules as rule}
					<button
						class="flex w-full items-center gap-2 py-0.5 pl-7 pr-3 text-left hover:bg-gray-800 {isSelected('rule', rule.data.id) ? 'bg-gray-800 text-white' : 'text-gray-400'}"
						onclick={() => select('rule', rule.data.id)}
					>
						<span class="truncate">{rule.data.id}</span>
					</button>
				{/each}
			{/if}
		</div>

		<!-- Events -->
		<div class="mt-2">
			<button
				class="flex w-full items-center gap-1 px-3 py-1 text-left text-xs font-semibold uppercase tracking-wider text-gray-500 hover:text-gray-300"
				onclick={() => toggle('events')}
			>
				<span class="w-3 text-center">{collapsed.events ? '▸' : '▾'}</span>
				<span>Events</span>
				<span class="ml-auto text-gray-600">{projectStore.project.events.length}</span>
			</button>

			{#if !collapsed.events}
				{#each projectStore.project.events as event}
					<button
						class="flex w-full items-center gap-2 py-0.5 pl-7 pr-3 text-left hover:bg-gray-800 {isSelected('event', event.data.event) ? 'bg-gray-800 text-white' : 'text-gray-400'}"
						onclick={() => select('event', event.data.event)}
					>
						<span class="truncate">{event.data.event}</span>
					</button>
				{/each}
			{/if}
		</div>
	</div>

	<!-- Add buttons -->
	<div class="border-t border-gray-800 p-2">
		<div class="grid grid-cols-2 gap-1">
			<button onclick={() => onadd('room')} class="rounded px-2 py-1 text-xs text-gray-400 hover:bg-gray-800 hover:text-gray-200">+ Room</button>
			<button onclick={() => onadd('entity')} class="rounded px-2 py-1 text-xs text-gray-400 hover:bg-gray-800 hover:text-gray-200">+ Entity</button>
			<button onclick={() => onadd('rule')} class="rounded px-2 py-1 text-xs text-gray-400 hover:bg-gray-800 hover:text-gray-200">+ Rule</button>
			<button onclick={() => onadd('event')} class="rounded px-2 py-1 text-xs text-gray-400 hover:bg-gray-800 hover:text-gray-200">+ Event</button>
		</div>
	</div>
</nav>
