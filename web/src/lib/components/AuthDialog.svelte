<script lang="ts">
	type SubmitData = { tailnet: string; apiKey: string };

	let {
		open = $bindable(false),
		initialTailnet = '',
		cloudBusy = false,
		cloudError = '',
		onClose = () => {},
		onSubmit
	}: {
		open?: boolean;
		initialTailnet?: string;
		cloudBusy?: boolean;
		cloudError?: string;
		onClose?: () => void;
		onSubmit?: (data: SubmitData) => void;
	} = $props();

	let authTailnet = $state('-');
	let authAPIKey = $state('');

	$effect(() => {
		if (open) {
			authTailnet = initialTailnet || '-';
		}
	});

	function handleSubmit(event: SubmitEvent & { currentTarget: EventTarget & HTMLFormElement }) {
		event.preventDefault();
		onSubmit?.({
			tailnet: authTailnet.trim() || '-',
			apiKey: authAPIKey
		});
	}

	function handleClose() {
		if (cloudBusy) return;
		onClose();
	}
</script>

{#if open}
	<div
		class="fixed inset-0 z-20 grid place-items-center bg-dialog-backdrop p-4"
		role="presentation"
	>
		<div
			class="w-[min(31rem,100%)] overflow-hidden rounded-lg border border-dialog-border bg-dialog-bg shadow-[0_24px_72px_rgb(23_33_38/24%)]"
			role="dialog"
			aria-modal="true"
			aria-labelledby="auth-title"
		>
			<div class="flex items-center justify-between gap-4 border-b border-panel-strong p-4">
				<div>
					<h2 id="auth-title" class="m-0 text-[1.1rem]">Enable ACL Editing</h2>
				</div>
				<button
					type="button"
					title="Close"
					onclick={handleClose}
					class="grid h-8 w-8 cursor-pointer place-items-center rounded-md border border-panel-border bg-panel-weak text-xl leading-none text-primary transition-[background-color,border-color] duration-[160ms] ease-out hover:border-teal hover:bg-hover"
					>×</button
				>
			</div>
			<form onsubmit={handleSubmit} class="flex flex-col gap-[0.8rem] p-4">
				<label class="flex flex-col gap-[0.35rem] text-[0.78rem] font-extrabold text-label">
					<span>Tailscale API Key</span>
					<input
						bind:value={authAPIKey}
						autocomplete="off"
						type="password"
						placeholder="tskey-api-..."
						class="min-h-[2.45rem] w-full rounded-md border border-dialog-border bg-dialog-input p-[0.5rem_0.65rem] text-[0.9rem] text-primary transition-[border-color,box-shadow] duration-[160ms] ease-out outline-none focus:border-teal focus:shadow-[0_0_0_3px_rgba(93,127,115,0.12)]"
					/>
				</label>
				<label class="flex flex-col gap-[0.35rem] text-[0.78rem] font-extrabold text-label">
					<span>Tailnet (optional)</span>
					<input
						bind:value={authTailnet}
						autocomplete="organization"
						placeholder="example.com or -"
						class="min-h-[2.45rem] w-full rounded-md border border-dialog-border bg-dialog-input p-[0.5rem_0.65rem] text-[0.9rem] text-primary transition-[border-color,box-shadow] duration-[160ms] ease-out outline-none focus:border-teal focus:shadow-[0_0_0_3px_rgba(93,127,115,0.12)]"
					/>
				</label>
				<p class="m-0 text-[0.84rem] text-tertiary">
					The backend uses this key to fetch the policy file and keeps it in memory only.
				</p>
				{#if cloudError}
					<p
						class="border-base-error m-0 rounded-md border bg-error p-[0.6rem_0.7rem] text-[0.84rem] font-bold text-error-text"
					>
						{cloudError}
					</p>
				{/if}
				<div class="flex justify-end gap-2 pt-[0.2rem]">
					<button class="btn-secondary" type="button" onclick={handleClose} disabled={cloudBusy}
						>Cancel</button
					>
					<button class="btn-primary" type="submit" disabled={cloudBusy}
						>{cloudBusy ? 'Connecting...' : 'Fetch Policy'}</button
					>
				</div>
			</form>
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
