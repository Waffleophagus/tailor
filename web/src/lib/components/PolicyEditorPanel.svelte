<script lang="ts">
	import type { PolicyResponse, StagedDraft } from '../api/schemas';

	let {
		open = $bindable(false),
		policy,
		editorText = $bindable(''),
		isDirty = false,
		valid = null,
		busy = false,
		status = '',
		errors = [] as string[],
		stagedDraft,
		onValidate = () => {},
		onSave = () => {},
		onDiscard = () => {},
		onClose = () => {}
	}: {
		open?: boolean;
		policy?: PolicyResponse;
		editorText?: string;
		isDirty?: boolean;
		valid?: boolean | null;
		busy?: boolean;
		status?: string;
		errors?: string[];
		stagedDraft?: StagedDraft;
		onValidate?: () => void;
		onSave?: () => void;
		onDiscard?: () => void;
		onClose?: () => void;
	} = $props();
</script>

{#if open && policy}
	<aside class="policy-editor" aria-label="Policy editor">
		<header class="editor-header">
			<div class="min-w-0">
				<p class="eyebrow">ACL policy</p>
				<h2 class="title">{policy.tailnet}</h2>
				<p class="hint">
					{#if stagedDraft}
						Review this {stagedDraft.source.toUpperCase()} draft, then save only if it matches your intent.
					{:else}
						Edit HuJSON directly. Validate with Tailscale, then save when the graph looks right.
					{/if}
				</p>
			</div>
			<button
				type="button"
				class="close-button"
				title="Close policy editor"
				aria-label="Close policy editor"
				onclick={onClose}>×</button
			>
		</header>

		<div class="editor-body">
			<textarea
				class="editor-textarea"
				bind:value={editorText}
				spellcheck="false"
				autocapitalize="off"
				autocomplete="off"
				aria-label="Policy HuJSON"
			></textarea>

			{#if errors.length > 0}
				<div class="error-panel" role="alert">
					{#each errors as error, index (index)}
						<p class="error-line">{error}</p>
					{/each}
				</div>
			{/if}
		</div>

		<footer class="editor-footer">
			<div class="footer-meta">
				{#if isDirty}
					<span class="dirty-pill">Unsaved changes</span>
				{/if}
				{#if stagedDraft}
					<span class="review-pill">Reviewing {stagedDraft.source}</span>
				{/if}
				{#if valid === true}
					<span class="valid-pill">Validated</span>
				{:else if valid === false}
					<span class="invalid-pill">Invalid</span>
				{/if}
				{#if status}
					<p class="status">{status}</p>
				{/if}
			</div>
			<div class="footer-actions">
				<button type="button" class="btn" onclick={onDiscard} disabled={busy || !isDirty}>
					Discard
				</button>
				<button type="button" class="btn" onclick={onValidate} disabled={busy || !isDirty}>
					Validate
				</button>
				<button
					type="button"
					class="btn primary"
					onclick={onSave}
					disabled={busy || valid !== true}
				>
					Save policy
				</button>
			</div>
		</footer>
	</aside>
{/if}

<style>
	@reference "../../app.css";

	.policy-editor {
		@apply absolute inset-y-3 right-3 z-[5] grid w-[min(42rem,calc(100%-1.5rem))] grid-rows-[auto_minmax(0,1fr)_auto] overflow-hidden rounded-xl border border-panel-border bg-panel-bg shadow-[0_18px_48px_rgb(23_33_38/16%)];
	}

	.editor-header {
		@apply flex items-start justify-between gap-3 border-b border-panel-strong px-4 py-3;
	}

	.eyebrow {
		@apply m-0 text-[0.72rem] font-extrabold tracking-wide text-secondary uppercase;
	}

	.title {
		@apply m-0 truncate font-extrabold text-base text-primary;
	}

	.hint {
		@apply mt-1 mb-0 text-[0.78rem] font-semibold text-secondary;
	}

	.close-button {
		@apply grid h-8 w-8 shrink-0 cursor-pointer place-items-center rounded-md border border-panel-border bg-panel-weak text-xl leading-none text-primary hover:border-teal hover:bg-hover;
	}

	.editor-body {
		@apply grid min-h-0 grid-rows-[minmax(0,1fr)_auto] gap-2 p-3;
	}

	.editor-textarea {
		@apply min-h-0 w-full resize-none rounded-lg border border-panel-border bg-[oklch(0.96_0.009_158)] p-3 font-mono text-[0.8rem] leading-[1.5] text-[#1c2c26] outline-none focus:border-teal;
	}

	.error-panel {
		@apply max-h-32 overflow-auto rounded-lg border border-danger/30 bg-panel-weak p-2;
	}

	.error-line {
		@apply m-0 text-[0.76rem] font-semibold text-danger;
	}

	.editor-footer {
		@apply flex flex-wrap items-end justify-between gap-3 border-t border-panel-strong px-4 py-3;
	}

	.footer-meta {
		@apply flex min-w-0 flex-wrap items-center gap-2;
	}

	.dirty-pill,
	.review-pill,
	.valid-pill,
	.invalid-pill {
		@apply rounded-full border px-2 py-[0.2rem] text-[0.72rem] font-extrabold uppercase;
	}

	.dirty-pill {
		@apply border-warn text-warn;
	}

	.review-pill {
		@apply border-teal text-teal;
	}

	.valid-pill {
		@apply border-ok text-ok;
	}

	.invalid-pill {
		@apply border-danger text-danger;
	}

	.status {
		@apply m-0 text-[0.76rem] font-semibold text-secondary;
	}

	.footer-actions {
		@apply flex shrink-0 flex-wrap gap-2;
	}

	.btn {
		@apply rounded-md border border-panel-border bg-panel-weak px-3 py-2 text-[0.82rem] font-extrabold text-primary disabled:cursor-not-allowed disabled:opacity-50;
	}

	.btn.primary {
		@apply border-panel-accent bg-panel-accent text-panel-fg;
	}
</style>
