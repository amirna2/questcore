<script lang="ts">
	import { projectStore } from '../../store/project.svelte.js';

	let { entityId }: { entityId: string } = $props();

	let entity = $derived(projectStore.project.entities.find((e) => e.data.id === entityId));
	let roomIds = $derived(projectStore.project.rooms.map((r) => r.data.id));

	function updateName(value: string) {
		projectStore.updateEntity(entityId, (e) => { e.name = value; });
	}

	function updateDescription(value: string) {
		projectStore.updateEntity(entityId, (e) => { e.description = value; });
	}

	function updateLocation(value: string) {
		projectStore.updateEntity(entityId, (e) => { e.location = value; });
	}

	function updateTakeable(value: boolean) {
		projectStore.updateEntity(entityId, (e) => {
			if (e.kind === 'item') e.takeable = value;
		});
	}

	function deleteEntity() {
		if (confirm(`Delete entity "${entityId}"?`)) {
			projectStore.removeEntity(entityId);
		}
	}
</script>

{#if entity}
	<div class="space-y-6">
		<div class="flex items-center justify-between">
			<h2 class="text-xl font-bold text-gray-100">
				<span class="text-gray-500 capitalize">{entity.data.kind}:</span> {entityId}
			</h2>
			<button
				onclick={deleteEntity}
				class="rounded px-3 py-1 text-xs text-red-400 hover:bg-red-900/30 hover:text-red-300"
			>
				Delete
			</button>
		</div>

		<div class="space-y-4">
			<label class="block">
				<span class="text-sm font-medium text-gray-400">Name</span>
				<input
					type="text"
					value={entity.data.name}
					oninput={(e) => updateName(e.currentTarget.value)}
					placeholder="Display name"
					class="mt-1 block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-gray-100 placeholder-gray-600 focus:border-blue-500 focus:outline-none"
				/>
			</label>

			<label class="block">
				<span class="text-sm font-medium text-gray-400">Description</span>
				<textarea
					value={entity.data.description}
					oninput={(e) => updateDescription(e.currentTarget.value)}
					placeholder="What does the player see when they examine this?"
					rows="3"
					class="mt-1 block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-gray-100 placeholder-gray-600 focus:border-blue-500 focus:outline-none"
				></textarea>
			</label>

			<label class="block">
				<span class="text-sm font-medium text-gray-400">Location</span>
				<select
					value={entity.data.location}
					onchange={(e) => updateLocation(e.currentTarget.value)}
					class="mt-1 block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
				>
					<option value="">— select a room —</option>
					{#each roomIds as id}
						<option value={id}>{id}</option>
					{/each}
				</select>
			</label>

			{#if entity.data.kind === 'item'}
				<label class="flex items-center gap-2">
					<input
						type="checkbox"
						checked={entity.data.takeable}
						onchange={(e) => updateTakeable(e.currentTarget.checked)}
						class="rounded border-gray-600 bg-gray-800 text-blue-500"
					/>
					<span class="text-sm text-gray-400">Takeable</span>
				</label>
			{/if}
		</div>
	</div>
{/if}
