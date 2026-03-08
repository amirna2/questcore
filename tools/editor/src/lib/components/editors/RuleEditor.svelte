<script lang="ts">
	import { projectStore } from '../../store/project.svelte.js';

	let { ruleId }: { ruleId: string } = $props();

	let rule = $derived(projectStore.project.rules.find((r) => r.data.id === ruleId));

	function updateVerb(value: string) {
		projectStore.updateRule(ruleId, (r) => { r.when.verb = value; });
	}

	function updateObject(value: string) {
		projectStore.updateRule(ruleId, (r) => {
			r.when.object = value || undefined;
		});
	}

	function updateTarget(value: string) {
		projectStore.updateRule(ruleId, (r) => {
			r.when.target = value || undefined;
		});
	}

	function deleteRule() {
		if (confirm(`Delete rule "${ruleId}"?`)) {
			projectStore.removeRule(ruleId);
		}
	}
</script>

{#if rule}
	<div class="space-y-6">
		<div class="flex items-center justify-between">
			<h2 class="text-xl font-bold text-gray-100">
				<span class="text-gray-500">Rule:</span> {ruleId}
			</h2>
			<button
				onclick={deleteRule}
				class="rounded px-3 py-1 text-xs text-red-400 hover:bg-red-900/30 hover:text-red-300"
			>
				Delete
			</button>
		</div>

		<!-- When clause -->
		<div>
			<h3 class="mb-2 text-sm font-semibold text-gray-300">When</h3>
			<div class="space-y-3 rounded border border-gray-700 bg-gray-800/50 p-4">
				<label class="block">
					<span class="text-xs text-gray-500">Verb</span>
					<input
						type="text"
						value={rule.data.when.verb}
						oninput={(e) => updateVerb(e.currentTarget.value)}
						placeholder="look, take, use..."
						class="mt-1 block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-100 placeholder-gray-600 focus:border-blue-500 focus:outline-none"
					/>
				</label>
				<label class="block">
					<span class="text-xs text-gray-500">Object (optional)</span>
					<input
						type="text"
						value={rule.data.when.object ?? ''}
						oninput={(e) => updateObject(e.currentTarget.value)}
						placeholder="entity or noun"
						class="mt-1 block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-100 placeholder-gray-600 focus:border-blue-500 focus:outline-none"
					/>
				</label>
				<label class="block">
					<span class="text-xs text-gray-500">Target (optional)</span>
					<input
						type="text"
						value={rule.data.when.target ?? ''}
						oninput={(e) => updateTarget(e.currentTarget.value)}
						placeholder="entity or noun"
						class="mt-1 block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-sm text-gray-100 placeholder-gray-600 focus:border-blue-500 focus:outline-none"
					/>
				</label>
			</div>
		</div>

		<!-- Conditions (placeholder for Phase 3) -->
		<div>
			<h3 class="mb-2 text-sm font-semibold text-gray-300">
				Conditions
				<span class="font-normal text-gray-600">({rule.data.conditions.length})</span>
			</h3>
			<div class="rounded border border-gray-700 bg-gray-800/50 p-4 text-sm text-gray-500">
				{#if rule.data.conditions.length > 0}
					{#each rule.data.conditions as cond}
						<div class="py-0.5 font-mono text-xs text-gray-400">{cond.type}: {JSON.stringify(cond)}</div>
					{/each}
				{:else}
					No conditions — rule always matches.
				{/if}
			</div>
		</div>

		<!-- Effects (placeholder for Phase 3) -->
		<div>
			<h3 class="mb-2 text-sm font-semibold text-gray-300">
				Effects
				<span class="font-normal text-gray-600">({rule.data.effects.length})</span>
			</h3>
			<div class="rounded border border-gray-700 bg-gray-800/50 p-4 text-sm text-gray-500">
				{#if rule.data.effects.length > 0}
					{#each rule.data.effects as effect, i}
						<div class="py-0.5 font-mono text-xs text-gray-400">{i + 1}. {effect.type}: {JSON.stringify(effect)}</div>
					{/each}
				{:else}
					No effects defined.
				{/if}
			</div>
		</div>
	</div>
{/if}
