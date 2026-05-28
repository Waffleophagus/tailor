<script lang="ts">
	import type { PolicyMapResponse } from '../api/schemas';
	import type { PolicyMutation } from '../draft/types';
	import GeneralAccessEditor from './editors/GeneralAccessEditor.svelte';
	import DefinitionEditor from './editors/DefinitionEditor.svelte';
	import SshEditor from './editors/SshEditor.svelte';
	import JsonSectionEditor from './editors/JsonSectionEditor.svelte';
	import WorkbenchSectionList from './WorkbenchSectionList.svelte';
	import type { WorkbenchRoute } from './nav';

	let {
		route,
		policyMap,
		search = $bindable(''),
		draftHuJSON = '',
		scenarioSource = '',
		editSeed = $bindable({ sources: '', destinations: '', ports: '443' }),
		busy = false,
		onViewAs = () => {},
		onMutate = () => {}
	}: {
		route: WorkbenchRoute;
		policyMap?: PolicyMapResponse;
		search?: string;
		draftHuJSON?: string;
		scenarioSource?: string;
		editSeed?: { sources: string; destinations: string; ports: string };
		busy?: boolean;
		onViewAs?: (selector: string) => void;
		onMutate?: (mutation: PolicyMutation, label: string) => void | Promise<void>;
	} = $props();

	const definitionRoutes: WorkbenchRoute[] = ['groups', 'tags', 'ip-sets', 'hosts'];
	const jsonRoutes: WorkbenchRoute[] = [
		'tests',
		'auto-approvers',
		'device-posture',
		'node-attributes',
		'advanced'
	];
</script>

{#if route === 'general-access'}
	<GeneralAccessEditor
		{policyMap}
		bind:search
		{draftHuJSON}
		{scenarioSource}
		bind:editSeed
		{busy}
		{onViewAs}
		{onMutate}
	/>
{:else if route === 'ssh'}
	<SshEditor {policyMap} bind:search {scenarioSource} {busy} {onMutate} />
{:else if definitionRoutes.includes(route)}
	<DefinitionEditor {route} {policyMap} bind:search {busy} {onViewAs} {onMutate} />
{:else if jsonRoutes.includes(route)}
	<JsonSectionEditor {route} {policyMap} bind:search {busy} {onMutate} />
{:else}
	<WorkbenchSectionList {route} {policyMap} bind:search {onViewAs} />
{/if}
