<script lang="ts">
	import { projectStore } from '../../store/project.svelte.js';

	let meta = $derived(projectStore.project.meta.data);
	let roomIds = $derived(projectStore.project.rooms.map((r) => r.data.id));

	function update<K extends keyof typeof meta>(field: K, value: (typeof meta)[K]) {
		projectStore.updateGameMeta((m) => {
			m[field] = value;
		});
	}
</script>

<div class="space-y-6">
	<h2 class="text-xl font-bold text-gray-100">Game Settings</h2>

	<div class="space-y-4">
		<label class="block">
			<span class="text-sm font-medium text-gray-400">Title</span>
			<input
				type="text"
				value={meta.title}
				oninput={(e) => update('title', e.currentTarget.value)}
				placeholder="My Adventure"
				class="mt-1 block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-gray-100 placeholder-gray-600 focus:border-blue-500 focus:outline-none"
			/>
		</label>

		<label class="block">
			<span class="text-sm font-medium text-gray-400">Author</span>
			<input
				type="text"
				value={meta.author}
				oninput={(e) => update('author', e.currentTarget.value)}
				placeholder="Your name"
				class="mt-1 block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-gray-100 placeholder-gray-600 focus:border-blue-500 focus:outline-none"
			/>
		</label>

		<label class="block">
			<span class="text-sm font-medium text-gray-400">Version</span>
			<input
				type="text"
				value={meta.version}
				oninput={(e) => update('version', e.currentTarget.value)}
				placeholder="1.0"
				class="mt-1 block w-full max-w-32 rounded border border-gray-700 bg-gray-800 px-3 py-2 text-gray-100 placeholder-gray-600 focus:border-blue-500 focus:outline-none"
			/>
		</label>

		<label class="block">
			<span class="text-sm font-medium text-gray-400">Start Room</span>
			<select
				value={meta.start}
				onchange={(e) => update('start', e.currentTarget.value)}
				class="mt-1 block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-gray-100 focus:border-blue-500 focus:outline-none"
			>
				<option value="">— select a room —</option>
				{#each roomIds as id}
					<option value={id}>{id}</option>
				{/each}
			</select>
		</label>

		<label class="block">
			<span class="text-sm font-medium text-gray-400">Intro Text</span>
			<textarea
				value={meta.intro}
				oninput={(e) => update('intro', e.currentTarget.value)}
				placeholder="The story begins..."
				rows="4"
				class="mt-1 block w-full rounded border border-gray-700 bg-gray-800 px-3 py-2 text-gray-100 placeholder-gray-600 focus:border-blue-500 focus:outline-none"
			></textarea>
		</label>
	</div>
</div>
