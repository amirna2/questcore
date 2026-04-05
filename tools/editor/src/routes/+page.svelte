<script lang="ts">
	import Toolbar from '$lib/components/Toolbar.svelte';
	import ExplorerTree from '$lib/components/ExplorerTree.svelte';
	import StatusBar from '$lib/components/StatusBar.svelte';
	import GameEditor from '$lib/components/editors/GameEditor.svelte';
	import RoomEditor from '$lib/components/editors/RoomEditor.svelte';
	import EntityEditor from '$lib/components/editors/EntityEditor.svelte';
	import RuleEditor from '$lib/components/editors/RuleEditor.svelte';
	import EventEditor from '$lib/components/editors/EventEditor.svelte';
	import { projectStore } from '$lib/store/project.svelte.js';
	import { importLuaFiles } from '$lib/parser/import.js';
	import { generateAll } from '$lib/codegen/index.js';

	let selection = $derived(projectStore.selection);

	// ── Add item dialogs ──────────────────────────────────────

	function handleAdd(kind: string) {
		const id = prompt(`Enter ${kind} ID (e.g., my_${kind}):`);
		if (!id) return;

		switch (kind) {
			case 'room':
				projectStore.addRoom(id);
				projectStore.select({ kind: 'room', id });
				break;
			case 'entity': {
				const entityKind = prompt('Entity type: item, npc, or enemy?', 'item') as 'item' | 'npc' | 'enemy' | null;
				if (!entityKind || !['item', 'npc', 'enemy'].includes(entityKind)) return;
				projectStore.addEntity(id, entityKind);
				projectStore.select({ kind: 'entity', id });
				break;
			}
			case 'rule':
				projectStore.addRule(id);
				projectStore.select({ kind: 'rule', id });
				break;
			case 'event':
				projectStore.addEvent(id);
				projectStore.select({ kind: 'event', id });
				break;
		}
	}

	// ── Import ────────────────────────────────────────────────

	async function handleImport() {
		const input = document.createElement('input');
		input.type = 'file';
		input.multiple = true;
		input.accept = '.lua';
		// Allow folder selection — user picks the game directory
		input.setAttribute('webkitdirectory', '');

		input.onchange = async () => {
			if (!input.files?.length) return;

			const files: Record<string, string> = {};
			for (const file of input.files) {
				// Only import .lua files from the selected folder
				if (!file.name.endsWith('.lua')) continue;
				files[file.name] = await file.text();
			}

			if (Object.keys(files).length === 0) {
				alert('No .lua files found in the selected folder.');
				return;
			}

			const result = importLuaFiles(files);
			projectStore.loadProject(result.project);

			// Auto-select Game Settings after import
			projectStore.select({ kind: 'game', id: null });

			const parts: string[] = [];
			if (result.summary.rooms > 0) parts.push(`${result.summary.rooms} rooms`);
			if (result.summary.entities > 0) parts.push(`${result.summary.entities} entities`);
			if (result.summary.rules > 0) parts.push(`${result.summary.rules} rules`);
			if (result.summary.events > 0) parts.push(`${result.summary.events} events`);

			let msg = `Imported: ${parts.join(', ')}.`;
			if (result.warnings.length > 0) {
				msg += `\n\n${result.warnings.length} warning(s):\n` + result.warnings.join('\n');
			}
			alert(msg);
		};

		input.click();
	}

	// ── Export ─────────────────────────────────────────────────

	function handleExport() {
		const generated = generateAll(projectStore.project);

		for (const [filename, content] of Object.entries(generated)) {
			if (!content.trim()) continue;
			const blob = new Blob([content], { type: 'text/plain' });
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = filename;
			a.click();
			URL.revokeObjectURL(url);
		}
	}
</script>

<div class="flex h-screen flex-col bg-gray-950 text-gray-100">
	<!-- Toolbar -->
	<Toolbar onimport={handleImport} onexport={handleExport} />

	<!-- Main workspace -->
	<div class="flex flex-1 overflow-hidden">
		<!-- Explorer -->
		<div class="w-56 shrink-0 border-r border-gray-800">
			<ExplorerTree onadd={handleAdd} />
		</div>

		<!-- Editor area -->
		<div class="flex-1 overflow-y-auto p-8">
			{#if selection?.kind === 'game'}
				<GameEditor />
			{:else if selection?.kind === 'room' && selection.id}
				<RoomEditor roomId={selection.id} />
			{:else if selection?.kind === 'entity' && selection.id}
				<EntityEditor entityId={selection.id} />
			{:else if selection?.kind === 'rule' && selection.id}
				<RuleEditor ruleId={selection.id} />
			{:else if selection?.kind === 'event' && selection.id}
				<EventEditor eventName={selection.id} />
			{:else}
				<div class="flex h-full items-center justify-center text-gray-600">
					<div class="text-center">
						<p class="text-lg">Select an item from the explorer</p>
						<p class="mt-1 text-sm">or import an existing game to get started</p>
					</div>
				</div>
			{/if}
		</div>
	</div>

	<!-- Status bar -->
	<StatusBar />
</div>
