<script lang="ts">
	import { onDestroy, untrack } from 'svelte';
	import type { Snippet } from 'svelte';

	let {
		position = 'left',
		defaultWidth = 16 * 16,
		open = true,
		children,
		collapsed
	}: {
		position?: 'left' | 'right';
		defaultWidth?: number;
		open?: boolean;
		children: Snippet;
		collapsed: Snippet;
	} = $props();

	const SIDEBAR_MIN = 12 * 16; // 12rem minimum
	const SIDEBAR_MAX = 30 * 16; // 30rem maximum
	const SIDEBAR_COLLAPSED = 2.75 * 16; // 2.75rem

	let sidebarWidth = $state(untrack(() => defaultWidth));
	let resizing = $state(false);

	let currentOnMove: ((e: PointerEvent) => void) | null = null;
	let currentOnUp: (() => void) | null = null;

	onDestroy(() => {
		resizing = false;
		if (currentOnMove) {
			window.removeEventListener('pointermove', currentOnMove);
			currentOnMove = null;
		}
		if (currentOnUp) {
			window.removeEventListener('pointerup', currentOnUp);
			currentOnUp = null;
		}
	});

	function startResize(event: PointerEvent) {
		if (!open) return;
		resizing = true;
		const startX = event.clientX;
		const startWidth = sidebarWidth;

		function onMove(e: PointerEvent) {
			const delta = position === 'left' ? e.clientX - startX : startX - e.clientX;
			sidebarWidth = Math.min(Math.max(startWidth + delta, SIDEBAR_MIN), SIDEBAR_MAX);
		}

		function onUp() {
			resizing = false;
			window.removeEventListener('pointermove', onMove);
			window.removeEventListener('pointerup', onUp);
			currentOnMove = null;
			currentOnUp = null;
		}

		currentOnMove = onMove;
		currentOnUp = onUp;
		window.addEventListener('pointermove', onMove);
		window.addEventListener('pointerup', onUp);
	}
</script>

<div
	class="sidebar transition-[width] relative flex shrink-0 flex-col overflow-hidden bg-sidebar duration-[220ms] ease-[cubic-bezier(0.4,0,0.2,1)]"
	data-position={position}
	data-open={open}
	data-resizing={resizing}
	style="width: {open ? sidebarWidth / 16 + 'rem' : SIDEBAR_COLLAPSED / 16 + 'rem'};"
>
	<div
		class="content flex min-h-0 min-w-0 flex-1 flex-col overflow-y-auto p-4 opacity-100 transition-opacity delay-[40ms] duration-160 ease-out"
	>
		{@render children()}
	</div>

	<div
		class="handle absolute top-0 z-20 h-full w-1 cursor-col-resize bg-transparent transition-colors duration-160 ease-out hover:bg-teal"
		class:left-0={position === 'right'}
		class:right-0={position === 'left'}
		role="separator"
		aria-label="Resize sidebar"
		data-resizing={resizing}
		onpointerdown={startResize}
	></div>

	<div
		class="icon-bar pointer-events-none absolute inset-0 flex flex-col items-center bg-sidebar px-1 py-2 opacity-0 transition-opacity duration-[140ms] ease-out"
		aria-hidden={open}
	>
		<div class="flex flex-col items-center gap-[0.4rem]">
			{@render collapsed()}
		</div>
	</div>
</div>

<style>
	@reference "../../app.css";
	.sidebar[data-resizing='true'] {
		@apply transition-none;
	}
	.sidebar[data-open='false'] {
		width: 2.75rem !important;
	}
	.sidebar[data-open='false'] .content {
		@apply pointer-events-none opacity-0 transition-[opacity];
		transition-delay: 0ms;
	}
	.sidebar[data-open='false'] .icon-bar {
		@apply pointer-events-auto opacity-100;
	}
	.handle[data-resizing='true'] {
		@apply bg-teal;
	}
</style>
