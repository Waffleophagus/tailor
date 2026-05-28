<script lang="ts">
	import type {
		PolicyResponse,
		PolicyEvaluateDraftResponse,
		PolicyMapResponse
	} from '../api/schemas';

	let {
		open = $bindable(false),
		policy,
		policyMap,
		search = $bindable(''),
		draftHuJSON = '',
		draftRuleText = '',
		draftEvaluation,
		draftValid = false,
		editBusy = false,
		editStatus = '',
		editSource = $bindable(''),
		editDestination = $bindable(''),
		editPortPreset = $bindable('443'),
		editCustomPorts = $bindable(''),
		cloudError = '',
		onClose = () => {},
		onDraft = () => {},
		onValidate = () => {},
		onSave = () => {},
		onViewAs = () => {}
	}: {
		open?: boolean;
		policy?: PolicyResponse;
		policyMap?: PolicyMapResponse;
		search?: string;
		draftHuJSON?: string;
		draftRuleText?: string;
		draftEvaluation?: PolicyEvaluateDraftResponse;
		draftValid?: boolean;
		editBusy?: boolean;
		editStatus?: string;
		editSource?: string;
		editDestination?: string;
		editPortPreset?: string;
		editCustomPorts?: string;
		cloudError?: string;
		onClose?: () => void;
		onDraft?: () => void;
		onValidate?: () => void;
		onSave?: () => void;
		onViewAs?: (selector: string) => void;
	} = $props();

	function canSimulateEntry(sectionName: string, label: string) {
		return (
			(sectionName === 'groups' && label.startsWith('group:')) ||
			(sectionName === 'tagOwners' && label.startsWith('tag:'))
		);
	}

	const sections = $derived(policyMap?.sections ?? []);
	const query = $derived(search.trim().toLowerCase());
	const filtered = $derived(
		query
			? sections.filter((section) => {
					const haystack = [
						section.name,
						section.description ?? '',
						...(section.entries ?? []).flatMap((entry) => [
							entry.label,
							entry.summary ?? '',
							...(entry.selectors ?? [])
						])
					]
						.join(' ')
						.toLowerCase();
					return haystack.includes(query);
				})
			: sections
	);

	function formatValue(value: unknown) {
		if (value === undefined || value === null) return '';
		if (typeof value === 'string') return value;
		return JSON.stringify(value);
	}

	function handleDraft(event: SubmitEvent & { currentTarget: EventTarget & HTMLFormElement }) {
		event.preventDefault();
		onDraft();
	}
</script>

{#if open && policy}
	<section
		class="absolute right-3 bottom-3 z-[4] grid max-h-[min(38rem,calc(100%-5rem))] w-[min(52rem,calc(100%-1.5rem))] grid-rows-[auto_minmax(0,1fr)] overflow-hidden rounded-lg border border-panel-border bg-panel-bg shadow-[0_18px_48px_rgb(23_33_38/16%)]"
		aria-label="Raw HuJSON policy"
	>
		<div
			class="flex items-center justify-between gap-3 border-b border-panel-strong px-[0.9rem] py-[0.8rem]"
		>
			<div>
				<p class="m-0 text-[0.8rem] font-bold tracking-normal text-secondary uppercase">
					Policy editor
				</p>
				<h2 class="m-0">{policy.tailnet}</h2>
			</div>
			<button
				type="button"
				title="Close policy panel"
				onclick={onClose}
				class="grid h-8 w-8 cursor-pointer place-items-center rounded-md border border-panel-border bg-panel-weak text-xl leading-none text-primary transition-[background-color,border-color] duration-[160ms] ease-out hover:border-teal hover:bg-hover"
				>×</button
			>
		</div>
		<div class="min-h-0 overflow-auto">
			<div class="border-b border-panel-strong bg-panel-weak">
				<div
					class="grid grid-cols-[minmax(10rem,1fr)_minmax(12rem,18rem)] items-center gap-3 p-[0.9rem]"
				>
					<div>
						<p class="m-0 text-[0.8rem] font-bold tracking-normal text-secondary uppercase">
							Policy workbench
						</p>
						<h3 class="m-0 text-base">
							{policyMap ? `${policyMap.sections.length} sections` : 'Loading sections'}
						</h3>
					</div>
					<input
						bind:value={search}
						placeholder="Search selectors or sections"
						class="min-h-[2.25rem] w-full rounded-md border border-panel-border bg-panel-input p-[0.45rem_0.55rem] text-primary transition-[border-color,box-shadow] duration-[140ms] ease-out outline-none focus:border-teal focus:shadow-[0_0_0_3px_rgba(93,127,115,0.12)]"
					/>
				</div>
				{#if policyMap?.parseError}
					<div
						class="mx-[0.9rem] mt-[0.75rem] mb-[0.9rem] rounded-md bg-[oklch(0.93_0.035_45)] p-[0.65rem_0.75rem] text-[0.82rem] text-[#5d2616]"
						role="alert"
					>
						{policyMap.parseError}
					</div>
				{:else if filtered.length === 0}
					<div
						class="mx-[0.9rem] mt-[0.75rem] mb-[0.9rem] rounded-md bg-empty p-[0.65rem_0.75rem] text-[0.82rem] text-code"
					>
						No policy sections match the current search.
					</div>
				{:else}
					<div class="grid max-h-[17rem] overflow-auto border-t border-panel-strong">
						{#each filtered as section (section.name)}
							<details class="border-b border-panel-border" open={section.name === 'acls'}>
								<summary
									class="grid cursor-pointer grid-cols-[minmax(0,1fr)_auto_auto] items-center gap-2 p-[0.65rem_0.9rem] text-[0.86rem] font-extrabold text-section"
								>
									<span>{section.description || section.name}</span>
									<span
										class="rounded-full border border-panel-border bg-panel-input px-[0.4rem] py-[0.15rem] text-[0.72rem] text-status-text"
										>{section.count}</span
									>
									{#if !section.supported}
										<span
											class="rounded-full border border-panel-border bg-panel-input px-[0.4rem] py-[0.15rem] text-[0.72rem] text-status-text"
											>unsupported</span
										>
									{/if}
								</summary>
								{#if section.entries?.length}
									<div class="grid gap-[0.45rem] px-[0.9rem] pb-[0.75rem]">
										{#each section.entries as entry (entry.label)}
											<article
												class="grid grid-cols-[minmax(0,1fr)] gap-[0.45rem] rounded-md border border-panel-border bg-panel-input p-[0.6rem]"
											>
												<div class="flex items-start justify-between gap-2">
													<div class="min-w-0">
														<strong class="block text-[0.84rem] text-primary">{entry.label}</strong>
														{#if entry.summary}
															<p class="mt-[0.2rem] text-[0.8rem] wrap-anywhere text-code">
																{entry.summary}
															</p>
														{/if}
													</div>
													{#if canSimulateEntry(section.name, entry.label)}
														<button
															type="button"
															class="shrink-0 cursor-pointer rounded border border-panel-border bg-panel-weak px-2 py-1 text-[0.72rem] font-extrabold text-teal hover:bg-hover"
															onclick={() => onViewAs(entry.label)}
														>
															View as
														</button>
													{/if}
												</div>
												{#if entry.selectors?.length}
													<div class="flex flex-wrap gap-[0.3rem]">
														{#each entry.selectors as selector (selector)}
															<span
																class="rounded-full bg-[oklch(0.93_0.018_165)] px-[0.36rem] py-[0.16rem] text-[0.72rem] font-bold text-selector"
																>{selector}</span
															>
														{/each}
													</div>
												{/if}
												{#if !entry.summary && entry.value !== undefined}
													<code class="text-[0.76rem] wrap-anywhere text-code"
														>{formatValue(entry.value)}</code
													>
												{/if}
											</article>
										{/each}
									</div>
								{:else}
									<div
										class="mx-[0.9rem] mt-[0.75rem] mb-[0.9rem] rounded-md bg-empty p-[0.65rem_0.75rem] text-[0.82rem] text-code"
									>
										This section is empty.
									</div>
								{/if}
							</details>
						{/each}
					</div>
				{/if}
			</div>
			<form
				class="grid grid-cols-[1fr_1fr_10rem] gap-[0.7rem] border-b border-panel-strong bg-panel-bg p-[0.9rem]"
				onsubmit={handleDraft}
			>
				<label class="flex flex-col gap-[0.35rem] text-[0.75rem] font-extrabold text-label">
					<span>Sources</span>
					<input
						bind:value={editSource}
						placeholder="alice@example.com, group:eng, tag:client"
						class="min-h-[2.3rem] w-full rounded-md border border-panel-border bg-panel-weak p-[0.45rem_0.55rem] text-[0.86rem] text-primary transition-[border-color,box-shadow] duration-[140ms] ease-out outline-none focus:border-teal focus:shadow-[0_0_0_3px_rgba(93,127,115,0.12)]"
					/>
				</label>
				<label class="flex flex-col gap-[0.35rem] text-[0.75rem] font-extrabold text-label">
					<span>Destination</span>
					<input
						bind:value={editDestination}
						placeholder="tag:web, db-host, 100.64.0.10"
						class="min-h-[2.3rem] w-full rounded-md border border-panel-border bg-panel-weak p-[0.45rem_0.55rem] text-[0.86rem] text-primary transition-[border-color,box-shadow] duration-[140ms] ease-out outline-none focus:border-teal focus:shadow-[0_0_0_3px_rgba(93,127,115,0.12)]"
					/>
				</label>
				<label class="flex flex-col gap-[0.35rem] text-[0.75rem] font-extrabold text-label">
					<span>Ports</span>
					<select
						bind:value={editPortPreset}
						class="min-h-[2.3rem] w-full rounded-md border border-panel-border bg-panel-weak p-[0.45rem_0.55rem] text-[0.86rem] text-primary transition-[border-color,box-shadow] duration-[140ms] ease-out outline-none focus:border-teal focus:shadow-[0_0_0_3px_rgba(93,127,115,0.12)]"
					>
						<option value="443">HTTPS 443</option>
						<option value="80,443">HTTP/S 80,443</option>
						<option value="22">SSH 22</option>
						<option value="*">All ports</option>
						<option value="custom">Custom</option>
					</select>
				</label>
				{#if editPortPreset === 'custom'}
					<label class="flex flex-col gap-[0.35rem] text-[0.75rem] font-extrabold text-label">
						<span>Custom ports</span>
						<input
							bind:value={editCustomPorts}
							placeholder="8080,8443"
							class="min-h-[2.3rem] w-full rounded-md border border-panel-border bg-panel-weak p-[0.45rem_0.55rem] text-[0.86rem] text-primary transition-[border-color,box-shadow] duration-[140ms] ease-out outline-none focus:border-teal focus:shadow-[0_0_0_3px_rgba(93,127,115,0.12)]"
						/>
					</label>
				{/if}
				<div class="col-span-full flex items-center justify-end gap-2">
					<button class="btn-secondary" type="submit" disabled={editBusy}>Draft rule</button>
					<button
						class="btn-secondary"
						type="button"
						onclick={onValidate}
						disabled={editBusy || !draftHuJSON}>Validate</button
					>
					<button
						class="btn-primary"
						type="button"
						onclick={onSave}
						disabled={editBusy || !draftValid}>Save</button
					>
				</div>
				{#if editStatus}
					<p class="col-span-full m-0 text-[0.82rem] font-bold text-status-text">{editStatus}</p>
				{/if}
			</form>
			{#if draftRuleText}
				<div class="border-b border-panel-strong bg-[oklch(0.965_0.01_158)] p-[0.9rem]">
					<p class="m-0 text-[0.8rem] font-bold tracking-normal text-secondary uppercase">
						Rule to append
					</p>
					<pre
						class="m-0 min-h-0 overflow-auto bg-[oklch(0.96_0.009_158)] p-[0.9rem] font-mono text-[0.78rem] leading-[1.5] whitespace-pre text-[#1c2c26]">{draftRuleText}</pre>
				</div>
			{/if}
			{#if draftEvaluation}
				<div class="grid gap-2 border-b border-panel-strong bg-[oklch(0.965_0.012_178)] p-[0.9rem]">
					<p class="m-0 text-[0.8rem] font-bold tracking-normal text-secondary uppercase">
						Impact preview
					</p>
					<div class="flex flex-wrap gap-[0.4rem]">
						<span class="impact-pill">+{draftEvaluation.added.length} added</span>
						<span class="impact-pill">{draftEvaluation.changed.length} changed</span>
						<span class="impact-pill">-{draftEvaluation.removed.length} removed</span>
						<span class="impact-pill">{draftEvaluation.unchanged.length} unchanged</span>
						{#if draftEvaluation.broadAccess.length}
							<span class="impact-pill warning">{draftEvaluation.broadAccess.length} broad</span>
						{/if}
						{#if draftEvaluation.unresolvedSelectors.length}
							<span class="impact-pill warning"
								>{draftEvaluation.unresolvedSelectors.length} unresolved selectors</span
							>
						{/if}
						{#if draftEvaluation.applicationGrants.length}
							<span class="impact-pill">{draftEvaluation.applicationGrants.length} app grants</span>
						{/if}
					</div>
					{#if draftEvaluation.unsupportedSections.length}
						<p class="m-0 text-[0.78rem] text-code">
							Unsupported sections in draft: {draftEvaluation.unsupportedSections.join(', ')}
						</p>
					{/if}
					{#if draftEvaluation.applicationGrants.length}
						<p class="m-0 text-[0.78rem] text-code">
							App capabilities: {draftEvaluation.applicationGrants
								.flatMap((grant) => grant.capabilities)
								.join(', ')}
						</p>
					{/if}
				</div>
			{/if}
			<details class="raw-policy" open={!draftHuJSON}>
				<summary
					class="cursor-pointer p-[0.7rem_0.9rem] text-[0.82rem] font-extrabold text-status-text"
					>{draftHuJSON ? 'Draft HuJSON' : 'Current HuJSON'}</summary
				>
				<pre
					class="m-0 min-h-0 overflow-auto bg-[oklch(0.96_0.009_158)] p-[0.9rem] font-mono text-[0.78rem] leading-[1.5] whitespace-pre text-[#1c2c26]">{draftHuJSON ||
						policy.hujson}</pre>
			</details>
		</div>
	</section>
{/if}

{#if cloudError}
	<div
		class="border-base-error absolute right-3 bottom-3 z-[5] max-w-[min(32rem,calc(100%-1.5rem))] rounded-lg border bg-error p-[0.65rem_0.75rem] text-[0.84rem] font-bold text-error-text shadow-[0_12px_32px_rgb(23_33_38/12%)]"
		role="alert"
	>
		{cloudError}
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
	.impact-pill {
		@apply rounded-full border border-panel-border bg-panel-input px-[0.5rem] py-[0.22rem] text-[0.74rem] font-extrabold text-status-text;
	}
	.impact-pill.warning {
		@apply border-[oklch(0.74_0.09_55)] bg-[oklch(0.93_0.035_55)] text-[#5d3616];
	}
	.raw-policy pre {
		margin: 0;
		padding: 0.9rem;
	}
</style>
