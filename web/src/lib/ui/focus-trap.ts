import type { Attachment } from 'svelte/attachments';

const focusableSelector = [
	'a[href]',
	'button:not([disabled])',
	'textarea:not([disabled])',
	'input:not([disabled])',
	'select:not([disabled])',
	'[tabindex]:not([tabindex="-1"])'
].join(',');

function focusableWithin(container: HTMLElement) {
	return Array.from(container.querySelectorAll<HTMLElement>(focusableSelector)).filter(
		(element) => element.offsetParent !== null
	);
}

/**
 * Modal focus handling for dialogs rendered inside `{#if open}`: moves focus into the
 * dialog on mount (first form field, else first focusable), keeps Tab cycling inside
 * it, calls `onEscape` on Escape, and restores focus to the opener on unmount.
 */
export function focusTrap(onEscape?: () => void): Attachment<HTMLElement> {
	return (container) => {
		const opener = document.activeElement instanceof HTMLElement ? document.activeElement : null;
		if (!container.hasAttribute('tabindex')) container.tabIndex = -1;

		const field = container.querySelector<HTMLElement>(
			'input:not([disabled]), textarea:not([disabled]), select:not([disabled])'
		);
		(field ?? focusableWithin(container)[0] ?? container).focus();

		// Listen on the document, not the container: focus can leave the dialog (a
		// backdrop click, or the focused button being removed from the DOM), and
		// keyboard events would then never reach the container.
		function handleKeydown(event: KeyboardEvent) {
			if (event.key === 'Escape') {
				event.preventDefault();
				onEscape?.();
				return;
			}
			if (event.key !== 'Tab') return;

			const elements = focusableWithin(container);
			if (elements.length === 0) {
				event.preventDefault();
				container.focus();
				return;
			}
			const first = elements[0];
			const last = elements[elements.length - 1];
			const active = document.activeElement;
			if (!container.contains(active)) {
				event.preventDefault();
				(event.shiftKey ? last : first).focus();
			} else if (event.shiftKey && active === first) {
				event.preventDefault();
				last.focus();
			} else if (!event.shiftKey && active === last) {
				event.preventDefault();
				first.focus();
			}
		}

		// Pull focus back if something outside the dialog receives it.
		function handleFocusIn(event: FocusEvent) {
			if (event.target instanceof Node && !container.contains(event.target)) {
				(focusableWithin(container)[0] ?? container).focus();
			}
		}

		document.addEventListener('keydown', handleKeydown, true);
		document.addEventListener('focusin', handleFocusIn, true);
		return () => {
			document.removeEventListener('keydown', handleKeydown, true);
			document.removeEventListener('focusin', handleFocusIn, true);
			if (opener?.isConnected) opener.focus();
		};
	};
}
