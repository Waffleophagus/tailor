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
			if (event.shiftKey && document.activeElement === first) {
				event.preventDefault();
				last.focus();
			} else if (!event.shiftKey && document.activeElement === last) {
				event.preventDefault();
				first.focus();
			}
		}

		container.addEventListener('keydown', handleKeydown);
		return () => {
			container.removeEventListener('keydown', handleKeydown);
			if (opener?.isConnected) opener.focus();
		};
	};
}
