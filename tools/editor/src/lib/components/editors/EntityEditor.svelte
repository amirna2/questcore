<script lang="ts">
	import { projectStore } from '../../store/project.svelte.js';
	import type { Topic } from '../../model/export-types.js';

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

	// ── NPC Topics ─────────────────────────────────────────────

	function addTopic() {
		const key = prompt('Topic key (e.g., greet, quest, hint):');
		if (!key) return;
		projectStore.updateEntity(entityId, (e) => {
			if (e.kind === 'npc') {
				e.topics[key] = { text: '', requires: [], effects: [] };
			}
		});
	}

	function updateTopicText(key: string, text: string) {
		projectStore.updateEntity(entityId, (e) => {
			if (e.kind === 'npc' && e.topics[key]) {
				e.topics[key].text = text;
			}
		});
	}

	function removeTopic(key: string) {
		projectStore.updateEntity(entityId, (e) => {
			if (e.kind === 'npc') {
				delete e.topics[key];
			}
		});
	}

	// ── Enemy Stats ────────────────────────────────────────────

	function updateStat(field: 'hp' | 'maxHp' | 'attack' | 'defense', value: number) {
		projectStore.updateEntity(entityId, (e) => {
			if (e.kind === 'enemy') e.stats[field] = value;
		});
	}

	function updateBehaviorWeight(index: number, weight: number) {
		projectStore.updateEntity(entityId, (e) => {
			if (e.kind === 'enemy') e.behavior[index].weight = weight;
		});
	}

	function addBehavior() {
		const action = prompt('Behavior action (e.g., attack, defend, flee):');
		if (!action) return;
		projectStore.updateEntity(entityId, (e) => {
			if (e.kind === 'enemy') {
				e.behavior.push({ action, weight: 50 });
			}
		});
	}

	function removeBehavior(index: number) {
		projectStore.updateEntity(entityId, (e) => {
			if (e.kind === 'enemy') e.behavior.splice(index, 1);
		});
	}

	function updateLootGold(value: number) {
		projectStore.updateEntity(entityId, (e) => {
			if (e.kind === 'enemy') e.loot.gold = value;
		});
	}

	function addLootItem() {
		const id = prompt('Item ID to drop:');
		if (!id) return;
		projectStore.updateEntity(entityId, (e) => {
			if (e.kind === 'enemy') {
				e.loot.items.push({ id, chance: 100 });
			}
		});
	}

	function updateLootChance(index: number, chance: number) {
		projectStore.updateEntity(entityId, (e) => {
			if (e.kind === 'enemy') e.loot.items[index].chance = chance;
		});
	}

	function removeLootItem(index: number) {
		projectStore.updateEntity(entityId, (e) => {
			if (e.kind === 'enemy') e.loot.items.splice(index, 1);
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
			<!-- Common fields: name, description, location -->
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

			<!-- Item: takeable -->
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

			<!-- NPC: topics -->
			{#if entity.data.kind === 'npc'}
				<div>
					<div class="flex items-center justify-between">
						<span class="text-sm font-medium text-gray-400">Topics</span>
						<button
							onclick={addTopic}
							class="rounded px-2 py-0.5 text-xs text-blue-400 hover:bg-blue-900/30"
						>
							+ Add Topic
						</button>
					</div>
					<div class="mt-2 space-y-3">
						{#each Object.entries(entity.data.topics) as [key, topic]}
							<div class="rounded border border-gray-700 bg-gray-800/50 p-3">
								<div class="mb-2 flex items-center justify-between">
									<span class="text-sm font-semibold text-gray-300">{key}</span>
									<button
										onclick={() => removeTopic(key)}
										class="text-xs text-red-400 hover:text-red-300"
									>
										&times; Remove
									</button>
								</div>
								<textarea
									value={topic.text}
									oninput={(e) => updateTopicText(key, e.currentTarget.value)}
									placeholder="What the NPC says..."
									rows="2"
									class="block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-100 placeholder-gray-600 focus:border-blue-500 focus:outline-none"
								></textarea>
								{#if topic.requires.length > 0}
									<div class="mt-2">
										<span class="text-xs text-gray-500">Requires:</span>
										{#each topic.requires as cond}
											<span class="ml-1 rounded bg-gray-700 px-1.5 py-0.5 font-mono text-xs text-gray-400">{cond.type}</span>
										{/each}
									</div>
								{/if}
								{#if topic.effects.length > 0}
									<div class="mt-1">
										<span class="text-xs text-gray-500">Effects:</span>
										{#each topic.effects as eff}
											<span class="ml-1 rounded bg-gray-700 px-1.5 py-0.5 font-mono text-xs text-gray-400">{eff.type}</span>
										{/each}
									</div>
								{/if}
							</div>
						{/each}
						{#if Object.keys(entity.data.topics).length === 0}
							<p class="text-xs text-gray-600">No topics — add conversation topics for this NPC.</p>
						{/if}
					</div>
				</div>
			{/if}

			<!-- Enemy: stats, behavior, loot -->
			{#if entity.data.kind === 'enemy'}
				<!-- Combat Stats -->
				<div>
					<span class="text-sm font-medium text-gray-400">Combat Stats</span>
					<div class="mt-2 grid grid-cols-2 gap-3">
						<label class="block">
							<span class="text-xs text-gray-500">HP</span>
							<input
								type="number"
								value={entity.data.stats.hp}
								oninput={(e) => updateStat('hp', Number(e.currentTarget.value))}
								class="mt-1 block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-100 focus:border-blue-500 focus:outline-none"
							/>
						</label>
						<label class="block">
							<span class="text-xs text-gray-500">Max HP</span>
							<input
								type="number"
								value={entity.data.stats.maxHp}
								oninput={(e) => updateStat('maxHp', Number(e.currentTarget.value))}
								class="mt-1 block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-100 focus:border-blue-500 focus:outline-none"
							/>
						</label>
						<label class="block">
							<span class="text-xs text-gray-500">Attack</span>
							<input
								type="number"
								value={entity.data.stats.attack}
								oninput={(e) => updateStat('attack', Number(e.currentTarget.value))}
								class="mt-1 block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-100 focus:border-blue-500 focus:outline-none"
							/>
						</label>
						<label class="block">
							<span class="text-xs text-gray-500">Defense</span>
							<input
								type="number"
								value={entity.data.stats.defense}
								oninput={(e) => updateStat('defense', Number(e.currentTarget.value))}
								class="mt-1 block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-100 focus:border-blue-500 focus:outline-none"
							/>
						</label>
					</div>
				</div>

				<!-- Behavior Weights -->
				<div>
					<div class="flex items-center justify-between">
						<span class="text-sm font-medium text-gray-400">Behavior</span>
						<button
							onclick={addBehavior}
							class="rounded px-2 py-0.5 text-xs text-blue-400 hover:bg-blue-900/30"
						>
							+ Add
						</button>
					</div>
					<div class="mt-2 space-y-2">
						{#each entity.data.behavior as bw, i}
							<div class="flex items-center gap-2">
								<span class="w-20 text-right text-xs text-gray-500">{bw.action}</span>
								<input
									type="range"
									min="0"
									max="100"
									value={bw.weight}
									oninput={(e) => updateBehaviorWeight(i, Number(e.currentTarget.value))}
									class="flex-1"
								/>
								<span class="w-10 text-right text-xs text-gray-400">{bw.weight}%</span>
								<button
									onclick={() => removeBehavior(i)}
									class="text-xs text-red-400 hover:text-red-300"
								>
									&times;
								</button>
							</div>
						{/each}
					</div>
				</div>

				<!-- Loot Table -->
				<div>
					<div class="flex items-center justify-between">
						<span class="text-sm font-medium text-gray-400">Loot</span>
						<button
							onclick={addLootItem}
							class="rounded px-2 py-0.5 text-xs text-blue-400 hover:bg-blue-900/30"
						>
							+ Add Item
						</button>
					</div>
					<div class="mt-2 space-y-2">
						<label class="flex items-center gap-2">
							<span class="w-20 text-right text-xs text-gray-500">Gold</span>
							<input
								type="number"
								min="0"
								value={entity.data.loot.gold}
								oninput={(e) => updateLootGold(Number(e.currentTarget.value))}
								class="w-24 rounded border border-gray-700 bg-gray-800 px-2 py-1 text-sm text-gray-100 focus:border-blue-500 focus:outline-none"
							/>
						</label>
						{#each entity.data.loot.items as lootItem, i}
							<div class="flex items-center gap-2">
								<span class="w-20 text-right font-mono text-xs text-gray-400">{lootItem.id}</span>
								<input
									type="number"
									min="0"
									max="100"
									value={lootItem.chance}
									oninput={(e) => updateLootChance(i, Number(e.currentTarget.value))}
									class="w-20 rounded border border-gray-700 bg-gray-800 px-2 py-1 text-sm text-gray-100 focus:border-blue-500 focus:outline-none"
								/>
								<span class="text-xs text-gray-500">% chance</span>
								<button
									onclick={() => removeLootItem(i)}
									class="text-xs text-red-400 hover:text-red-300"
								>
									&times;
								</button>
							</div>
						{/each}
						{#if entity.data.loot.items.length === 0}
							<p class="ml-22 text-xs text-gray-600">No item drops.</p>
						{/if}
					</div>
				</div>
			{/if}
		</div>
	</div>
{/if}
