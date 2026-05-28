<script lang="ts">
	import type { PolicyResponse } from '../api/schemas';

	let {
		open = $bindable(false),
		policy,
		draftHuJSON = '',
		onClose = () => {}
	}: {
		open?: boolean;
		policy?: PolicyResponse;
		draftHuJSON?: string;
		onClose?: () => void;
	} = $props();
</script>

{#if open && policy}
	<section
		class="absolute right-3 bottom-3 z-[5] grid max-h-[min(28rem,calc(100%-5rem))] w-[min(36rem,calc(100%-1.5rem))] grid-rows-[auto_minmax(0,1fr)] overflow-hidden rounded-lg border border-panel-border bg-panel-bg shadow-[0_18px_48px_rgb(23_33_38/16%)]"
		aria-label="Raw HuJSON policy"
	>
		<div
			class="flex items-center justify-between gap-3 border-b border-panel-strong px-[0.9rem] py-[0.8rem]"
		>
			<div>
				<p class="m-0 text-[0.8rem] font-bold tracking-normal text-secondary uppercase">
					Raw HuJSON
				</p>
				<h2 class="m-0 text-base">{policy.tailnet}</h2>
			</div>
			<button type="button" class="close-button" title="Close raw policy" onclick={onClose}
				>×</button
			>
		</div>
		<pre
			class="m-0 min-h-0 overflow-auto bg-[oklch(0.96_0.009_158)] p-[0.9rem] font-mono text-[0.78rem] leading-[1.5] whitespace-pre text-[#1c2c26]">{draftHuJSON ||
				policy.hujson}</pre>
	</section>
{/if}

<style>
	@reference "../../app.css";

	.close-button {
		@apply grid h-8 w-8 cursor-pointer place-items-center rounded-md border border-panel-border bg-panel-weak text-xl leading-none text-primary transition-[background-color,border-color] duration-[160ms] hover:border-teal hover:bg-hover;
	}
</style>
