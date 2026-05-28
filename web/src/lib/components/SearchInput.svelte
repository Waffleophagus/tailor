<script lang="ts">
	let {
		value = $bindable(''),
		placeholder = 'Search...',
		count = 0,
		total = 0
	}: {
		value?: string;
		placeholder?: string;
		count?: number;
		total?: number;
	} = $props();
</script>

<div class="relative flex items-center">
	<svg
		class="pointer-events-none absolute left-[0.55rem] h-[0.9rem] w-[0.9rem] text-secondary"
		viewBox="0 0 20 20"
		fill="none"
		xmlns="http://www.w3.org/2000/svg"
	>
		<path
			d="M17.5 17.5L13.875 13.875M15.8333 9.16667C15.8333 12.8486 12.8486 15.8333 9.16667 15.8333C5.48477 15.8333 2.5 12.8486 2.5 9.16667C2.5 5.48477 5.48477 2.5 9.16667 2.5C12.8486 2.5 15.8333 5.48477 15.8333 9.16667Z"
			stroke="currentColor"
			stroke-width="1.6"
			stroke-linecap="round"
			stroke-linejoin="round"
		/>
	</svg>
	<input
		type="text"
		{placeholder}
		bind:value
		aria-label="Search devices"
		class="h-8 w-full rounded-md border border-medium bg-input p-[0_1.75rem_0_1.7rem] text-sm text-primary transition-[border-color,box-shadow] duration-[140ms] ease-out outline-none placeholder:text-muted focus:border-teal focus:shadow-[0_0_0_3px_rgba(93,127,115,0.12)]"
	/>
	{#if value}
		<button class="clear" type="button" title="Clear search" onclick={() => (value = '')}>
			<svg viewBox="0 0 14 14" fill="none" xmlns="http://www.w3.org/2000/svg">
				<path
					d="M1 1L13 13M13 1L1 13"
					stroke="currentColor"
					stroke-width="1.4"
					stroke-linecap="round"
				/>
			</svg>
		</button>
	{/if}
</div>
<p class="mt-1.5 mb-0 text-[0.78rem] leading-[1.3] font-bold text-tertiary">
	{count} of {total} devices visible
</p>

<style>
	@reference "../../app.css";
	.clear {
		@apply absolute right-[0.35rem] grid h-[1.4rem] w-[1.4rem] place-items-center rounded border-none bg-transparent p-0 text-muted transition-colors duration-120 ease-out;
	}
	.clear:hover {
		@apply text-secondary;
	}
	.clear svg {
		@apply h-[0.7rem] w-[0.7rem];
	}
</style>
