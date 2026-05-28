<script lang="ts">
	let {
		open = true,
		position = 'left',
		ontoggle
	}: {
		open?: boolean;
		position?: 'left' | 'right';
		ontoggle?: () => void;
	} = $props();

	const arrow = $derived(
		open ? (position === 'left' ? '←' : '→') : position === 'left' ? '→' : '←'
	);
</script>

<button
	class="sidebar-toggle absolute top-2 z-10 grid h-[1.7rem] w-[1.7rem] cursor-pointer place-items-center rounded-md border border-medium bg-surface p-0 text-xs leading-none font-bold text-secondary shadow-[0_2px_6px_rgba(23,33,38,0.06)] transition-[background-color,border-color,color,transform] duration-[160ms] ease-out hover:scale-105 hover:border-teal hover:bg-hover hover:text-primary motion-reduce:transition-none"
	data-position={position}
	title={open ? 'Collapse panel' : 'Expand panel'}
	type="button"
	onclick={ontoggle}
>
	{arrow}
</button>

<style>
	.sidebar-toggle[data-position='left'] {
		left: -0.85rem;
	}
	.sidebar-toggle[data-position='right'] {
		right: -0.85rem;
	}
</style>
