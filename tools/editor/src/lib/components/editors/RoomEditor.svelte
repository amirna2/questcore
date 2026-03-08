<script lang="ts">
	import { projectStore } from '../../store/project.svelte.js';
	import type { Direction } from '../../model/export-types.js';

	let { roomId }: { roomId: string } = $props();

	let room = $derived(projectStore.project.rooms.find((r) => r.data.id === roomId));
	let allRoomIds = $derived(projectStore.project.rooms.map((r) => r.data.id));

	const DIRECTIONS: Direction[] = [
		'north', 'south', 'east', 'west',
		'northeast', 'northwest', 'southeast', 'southwest',
		'up', 'down'
	];

	function updateDescription(value: string) {
		projectStore.updateRoom(roomId, (r) => { r.description = value; });
	}

	function setExit(dir: Direction, target: string) {
		projectStore.updateRoom(roomId, (r) => {
			if (target) {
				r.exits[dir] = target;
			} else {
				delete r.exits[dir];
			}
		});
	}

	function deleteRoom() {
		if (confirm(`Delete room "${roomId}"?`)) {
			projectStore.removeRoom(roomId);
		}
	}
</script>

{#if room}
	<div class="space-y-6">
		<div class="flex items-center justify-between">
			<h2 class="text-xl font-bold text-gray-100">
				<span class="text-gray-500">Room:</span> {roomId}
			</h2>
			<button
				onclick={deleteRoom}
				class="rounded px-3 py-1 text-xs text-red-400 hover:bg-red-900/30 hover:text-red-300"
			>
				Delete
			</button>
		</div>

		<label class="block">
			<span class="text-sm font-medium text-gray-400">Description</span>
			<textarea
				value={room.data.description}
				oninput={(e) => updateDescription(e.currentTarget.value)}
				placeholder="Describe what the player sees..."
				rows="4"
				class="mt-1 block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-gray-100 placeholder-gray-600 focus:border-blue-500 focus:outline-none"
			></textarea>
		</label>

		<!-- Exits -->
		<div>
			<span class="text-sm font-medium text-gray-400">Exits</span>
			<div class="mt-2 grid gap-2">
				{#each DIRECTIONS as dir}
					<div class="flex items-center gap-2">
						<span class="w-24 text-right text-xs text-gray-500">{dir}</span>
						<select
							value={room.data.exits[dir] ?? ''}
							onchange={(e) => setExit(dir, e.currentTarget.value)}
							class="flex-1 rounded border border-gray-700 bg-gray-800 px-2 py-1 text-sm text-gray-100 focus:border-blue-500 focus:outline-none"
						>
							<option value="">—</option>
							{#each allRoomIds.filter((id) => id !== roomId) as id}
								<option value={id}>{id}</option>
							{/each}
						</select>
					</div>
				{/each}
			</div>
		</div>
	</div>
{/if}
