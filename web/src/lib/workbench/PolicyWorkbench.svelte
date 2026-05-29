<script lang="ts">
	import type { PolicyMapResponse, PolicyResponse } from '../api/schemas';
	import type { PolicyMutation } from '../draft/types';
	import WorkbenchSectionView from './WorkbenchSectionView.svelte';
	import {
		WORKBENCH_NAV,
		defaultWorkbenchRoute,
		hasAdvancedSections,
		simulationTierLabel,
		type WorkbenchRoute
	} from './nav';

	let {
		open = $bindable(false),
		route = $bindable(defaultWorkbenchRoute()),
		policy,
		policyMap,
		search = $bindable(''),
		draftHuJSON = '',
		scenarioSource = '',
		editSeed = $bindable({ sources: '', destinations: '', ports: '443' }),
		editBusy = false,
		activeScenarioLabel = '',
		onClose = () => {},
		onViewAs = () => {},
		onMutate = () => {}
	}: {
		open?: boolean;
		route?: WorkbenchRoute;
		policy?: PolicyResponse;
		policyMap?: PolicyMapResponse;
		search?: string;
		draftHuJSON?: string;
		scenarioSource?: string;
		editSeed?: { sources: string; destinations: string; ports: string };
		editBusy?: boolean;
		activeScenarioLabel?: string;
		onClose?: () => void;
		onViewAs?: (selector: string) => void;
		onMutate?: (mutation: PolicyMutation, label: string) => void | Promise<void>;
	} = $props();

	const policyItems = $derived(WORKBENCH_NAV.filter((item) => item.group === 'policy'));
	const definitionItems = $derived(WORKBENCH_NAV.filter((item) => item.group === 'definitions'));
	const showAdvanced = $derived(hasAdvancedSections(policyMap));

	function selectRoute(id: WorkbenchRoute) {
		route = id;
		search = '';
	}
</script>

{#if open && policy}
	<aside
		class="workbench shrink-0 overflow-hidden border-l border-panel-border bg-panel-bg shadow-[-12px_0_32px_rgb(23_33_38/8%)]"
		aria-label="Access controls"
	>
		<div class="grid h-full min-h-0 grid-rows-[auto_minmax(0,1fr)]">
			<header
				class="flex items-center justify-between gap-3 border-b border-panel-strong px-4 py-3"
			>
				<div class="min-w-0">
					<p class="m-0 text-[0.72rem] font-extrabold tracking-wide text-secondary uppercase">
						Access controls
					</p>
					<h1 class="m-0 truncate font-extrabold text-base text-primary">{policy.tailnet}</h1>
					{#if activeScenarioLabel}
						<p class="mt-1 mb-0 truncate text-[0.76rem] text-teal">{activeScenarioLabel}</p>
					{/if}
				</div>
				<button
					type="button"
					class="close-button"
					title="Close access controls"
					aria-label="Close access controls"
					onclick={onClose}>×</button
				>
			</header>

			<div class="grid min-h-0 grid-cols-[11.5rem_minmax(0,1fr)]">
				<nav class="min-h-0 overflow-y-auto border-r border-panel-strong bg-panel-weak p-2">
					<p class="nav-heading">Policy</p>
					<ul class="nav-list">
						{#each policyItems as item (item.id)}
							<li>
								<button
									type="button"
									class="nav-button"
									data-active={route === item.id}
									onclick={() => selectRoute(item.id)}
								>
									<span>{item.label}</span>
									{#if item.simulationTier !== 'graph-simulated'}
										<span class="tier-badge">{simulationTierLabel(item.simulationTier)}</span>
									{/if}
								</button>
							</li>
						{/each}
					</ul>
					<p class="nav-heading">Definitions</p>
					<ul class="nav-list">
						{#each definitionItems as item (item.id)}
							<li>
								<button
									type="button"
									class="nav-button"
									data-active={route === item.id}
									onclick={() => selectRoute(item.id)}
								>
									<span>{item.label}</span>
									{#if item.simulationTier !== 'graph-simulated'}
										<span class="tier-badge">{simulationTierLabel(item.simulationTier)}</span>
									{/if}
								</button>
							</li>
						{/each}
						{#if showAdvanced}
							<li>
								<button
									type="button"
									class="nav-button"
									data-active={route === 'advanced'}
									onclick={() => selectRoute('advanced')}>Advanced</button
								>
							</li>
						{/if}
					</ul>
				</nav>

				<WorkbenchSectionView
					{route}
					{policyMap}
					bind:search
					{draftHuJSON}
					{scenarioSource}
					bind:editSeed
					busy={editBusy}
					{onViewAs}
					{onMutate}
				/>
			</div>
		</div>
	</aside>
{/if}

<style>
	@reference "../../app.css";
	.workbench {
		width: min(44rem, 46vw);
		animation: slide-in 160ms cubic-bezier(0.4, 0, 0.2, 1);
	}
	@keyframes slide-in {
		from {
			opacity: 0.92;
			transform: translateX(1.25rem);
		}
		to {
			opacity: 1;
			transform: translateX(0);
		}
	}
	.close-button {
		@apply grid h-8 w-8 shrink-0 cursor-pointer place-items-center rounded-md border border-panel-border bg-panel-weak text-xl leading-none text-primary hover:border-teal hover:bg-hover;
	}
	.nav-heading {
		@apply m-0 px-2 pt-3 pb-1 text-[0.68rem] font-extrabold tracking-[0.08em] text-label uppercase;
	}
	.nav-list {
		@apply m-0 list-none p-0;
	}
	.nav-button {
		@apply mb-[0.15rem] flex w-full cursor-pointer flex-col items-start gap-[0.15rem] rounded-md border-0 bg-transparent px-2 py-[0.45rem] text-left text-[0.8rem] font-bold text-secondary hover:bg-hover hover:text-primary;
	}
	.nav-button[data-active='true'] {
		@apply bg-hover text-primary;
	}
	.tier-badge {
		@apply text-[0.62rem] font-extrabold tracking-normal text-label uppercase;
	}
</style>
