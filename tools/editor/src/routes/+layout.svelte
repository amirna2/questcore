<script>
	import '../app.css';
	let { children } = $props();
</script>

<div class="flex min-h-screen bg-gray-950 text-gray-100">
	<!-- Sidebar placeholder — Phase 2 -->
	<nav class="w-56 shrink-0 border-r border-gray-800 bg-gray-900 p-4">
		<h1 class="mb-6 text-lg font-bold tracking-tight">QuestCore Editor</h1>
		<p class="text-sm text-gray-500">Coming soon...</p>
	</nav>

	<!-- Main content -->
	<main class="flex-1 p-8">
		{@render children()}
	</main>
</div>
