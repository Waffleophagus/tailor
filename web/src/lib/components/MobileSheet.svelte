<script lang="ts">
	import { tick, type Snippet } from 'svelte';

	let {
		open = false,
		title = '',
		onclose,
		children
	}: {
		open?: boolean;
		title?: string;
		onclose?: () => void;
		children: Snippet;
	} = $props();

	let sheetElement = $state<HTMLDivElement>();
	let previouslyFocused: HTMLElement | undefined;
	let wasOpen = false;

	const focusableSelector = [
		'a[href]',
		'button:not([disabled])',
		'textarea:not([disabled])',
		'input:not([disabled])',
		'select:not([disabled])',
		'[tabindex]:not([tabindex="-1"])'
	].join(',');

	function focusableElements() {
		if (!sheetElement) return [];
		return Array.from(sheetElement.querySelectorAll<HTMLElement>(focusableSelector)).filter(
			(element) => !element.hasAttribute('disabled') && element.offsetParent !== null
		);
	}

	function focusSheet() {
		const [firstFocusable] = focusableElements();
		(firstFocusable ?? sheetElement)?.focus();
	}

	function restoreFocus() {
		previouslyFocused?.focus();
		previouslyFocused = undefined;
	}

	function handleKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			event.preventDefault();
			close();
			return;
		}

		if (event.key !== 'Tab') return;

		const elements = focusableElements();
		if (elements.length === 0) {
			event.preventDefault();
			sheetElement?.focus();
			return;
		}

		const first = elements[0];
		const last = elements[elements.length - 1];

		if (event.shiftKey && document.activeElement === first) {
			event.preventDefault();
			last.focus();
		} else if (!event.shiftKey && document.activeElement === last) {
			event.preventDefault();
			first.focus();
		} else if (!sheetElement?.contains(document.activeElement)) {
			event.preventDefault();
			first.focus();
		}
	}

	function close() {
		onclose?.();
	}

	$effect(() => {
		if (open && !wasOpen) {
			previouslyFocused =
				document.activeElement instanceof HTMLElement ? document.activeElement : undefined;
			wasOpen = true;
			tick().then(() => {
				if (open) focusSheet();
			});
		} else if (!open && wasOpen) {
			wasOpen = false;
			restoreFocus();
		}
	});
</script>

{#if open}
	<button
		type="button"
		class="backdrop"
		aria-label="Close panel"
		onclick={close}
		onkeydown={handleKeydown}
		tabindex="-1"
	></button>
	<div
		bind:this={sheetElement}
		class="sheet"
		role="dialog"
		aria-modal="true"
		aria-label={title || 'Panel'}
		tabindex="-1"
		onkeydown={handleKeydown}
	>
		<div class="handle" aria-hidden="true"></div>
		{#if title}
			<div class="header">
				<h2 class="title">{title}</h2>
				<button type="button" class="close" aria-label="Close" onclick={close}>×</button>
			</div>
		{/if}
		<div class="body">
			{@render children()}
		</div>
	</div>
{/if}

<style>
	@reference "../../app.css";
	.backdrop {
		@apply fixed inset-0 z-[15] border-0 bg-dialog-backdrop p-0;
	}
	.sheet {
		@apply fixed right-0 bottom-0 left-0 z-[16] flex max-h-[min(78vh,calc(100%-4rem))] flex-col rounded-t-2xl border border-graph-border bg-surface shadow-[0_-12px_40px_rgb(23_33_38/12%)];
		padding-bottom: env(safe-area-inset-bottom, 0px);
		animation: sheet-enter 280ms cubic-bezier(0.16, 1, 0.3, 1);
	}
	@keyframes sheet-enter {
		from {
			transform: translateY(100%);
		}
		to {
			transform: translateY(0);
		}
	}
	@media (prefers-reduced-motion: reduce) {
		.sheet {
			animation: none;
		}
	}
	.handle {
		@apply mx-auto mt-2 h-1 w-10 shrink-0 rounded-full bg-medium;
	}
	.header {
		@apply flex shrink-0 items-center justify-between gap-3 border-b border-light px-4 py-3;
	}
	.title {
		@apply m-0 text-[0.95rem] leading-[1.2] font-extrabold text-primary;
	}
	.close {
		@apply grid h-11 w-11 shrink-0 cursor-pointer place-items-center rounded-md border border-panel-border bg-panel-weak text-xl leading-none font-bold text-secondary transition-[background-color,border-color,color] duration-[140ms] ease-out hover:border-teal hover:bg-hover hover:text-primary;
	}
	.body {
		@apply min-h-0 flex-1 overflow-y-auto px-4 py-3;
	}
</style>
