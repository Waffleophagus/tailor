<script lang="ts">
	let {
		open = $bindable(false),
		snippet = '',
		statusMessage = '',
		busy = false,
		error = '',
		onClose = () => {},
		onAccept = () => {},
		onSaveEdited = () => {}
	}: {
		open?: boolean;
		snippet?: string;
		statusMessage?: string;
		busy?: boolean;
		error?: string;
		onClose?: () => void;
		onAccept?: () => void;
		onSaveEdited?: (snippet: string) => void;
	} = $props();

	let editing = $state(false);
	let editedSnippet = $state('');

	$effect(() => {
		if (open) {
			editing = false;
			editedSnippet = snippet;
		}
	});

	function handleClose() {
		if (busy) return;
		onClose();
	}
</script>

{#if open}
	<div
		class="fixed inset-0 z-20 grid place-items-center bg-dialog-backdrop p-4"
		role="presentation"
	>
		<div
			class="w-[min(40rem,100%)] overflow-hidden rounded-lg border border-dialog-border bg-dialog-bg shadow-[0_24px_72px_rgb(23_33_38/24%)]"
			role="dialog"
			aria-modal="true"
			aria-labelledby="setup-grant-title"
		>
			<div class="flex items-center justify-between gap-4 border-b border-panel-strong p-4">
				<div>
					<h2 id="setup-grant-title" class="m-0 text-[1.1rem]">Configure Tailor Access</h2>
				</div>
				<button
					type="button"
					title="Close"
					onclick={handleClose}
					class="grid h-8 w-8 cursor-pointer place-items-center rounded-md border border-panel-border bg-panel-weak text-xl leading-none text-primary transition-[background-color,border-color] duration-[160ms] ease-out hover:border-teal hover:bg-hover"
					>×</button
				>
			</div>
			<div class="flex flex-col gap-[0.8rem] p-4">
				{#if statusMessage}
					<p class="m-0 text-[0.9rem] text-secondary">{statusMessage}</p>
				{/if}
				{#if editing}
					<label class="flex flex-col gap-[0.35rem] text-[0.78rem] font-extrabold text-label">
						<span>App capability grant</span>
						<textarea
							bind:value={editedSnippet}
							rows="14"
							spellcheck="false"
							class="w-full rounded-md border border-dialog-border bg-dialog-input p-[0.65rem] font-mono text-[0.82rem] text-primary outline-none focus:border-teal focus:shadow-[0_0_0_3px_rgba(93,127,115,0.12)]"
						></textarea>
					</label>
				{:else}
					<pre
						class="m-0 overflow-x-auto rounded-md border border-panel-border bg-panel-weak p-[0.65rem] font-mono text-[0.78rem] whitespace-pre-wrap text-primary">{snippet}</pre>
				{/if}
				{#if error}
					<p
						class="border-base-error m-0 rounded-md border bg-error p-[0.6rem_0.7rem] text-[0.84rem] font-bold text-error-text"
					>
						{error}
					</p>
				{/if}
				<div class="flex flex-wrap justify-end gap-2 pt-[0.2rem]">
					<button class="btn-secondary" type="button" onclick={handleClose} disabled={busy}
						>Not now</button
					>
					{#if editing}
						<button
							class="btn-secondary"
							type="button"
							disabled={busy}
							onclick={() => (editing = false)}>Back</button
						>
						<button
							class="btn-primary"
							type="button"
							disabled={busy}
							onclick={() => onSaveEdited(editedSnippet)}
						>
							{busy ? 'Saving...' : 'Save edited grant'}
						</button>
					{:else}
						<button
							class="btn-secondary"
							type="button"
							disabled={busy}
							onclick={() => (editing = true)}>Edit grant</button
						>
						<button class="btn-primary" type="button" disabled={busy} onclick={onAccept}>
							{busy ? 'Applying...' : 'Add recommended grant'}
						</button>
					{/if}
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	@reference "../../app.css";
	.btn-primary {
		@apply min-h-[2.35rem] rounded-md border border-panel-accent bg-panel-accent px-3 py-[0.45rem] text-sm font-extrabold text-panel-fg transition-[background-color,border-color,color,transform] duration-[160ms] ease-out hover:-translate-y-px disabled:transform-none disabled:cursor-not-allowed disabled:opacity-[0.58];
	}
	.btn-secondary {
		@apply min-h-[2.35rem] rounded-md border border-panel-border bg-panel-weak px-3 py-[0.45rem] text-sm font-extrabold text-primary transition-[background-color,border-color,color,transform] duration-[160ms] ease-out hover:-translate-y-px disabled:transform-none disabled:cursor-not-allowed disabled:opacity-[0.58];
	}
</style>
