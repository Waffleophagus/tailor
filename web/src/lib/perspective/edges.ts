import type { RenderEdge } from '../graph/engine';

function mergeKey(edge: RenderEdge) {
	return `${edge.from}\0${edge.to}`;
}

function mergeEdges(existing: RenderEdge, incoming: RenderEdge): RenderEdge {
	const ports = new Set([...(existing.ports ?? []), ...(incoming.ports ?? [])]);
	const protocols = new Set([...(existing.protocols ?? []), ...(incoming.protocols ?? [])]);
	const perspectives = new Set([
		...(existing.perspectives ?? []),
		...(incoming.perspectives ?? [])
	]);
	const policyRefs = [...(existing.policyRefs ?? []), ...(incoming.policyRefs ?? [])];
	return {
		...existing,
		ports: [...ports].sort(),
		protocols: [...protocols].sort(),
		perspectives: [...perspectives].sort(),
		policyRefs,
		state: incoming.state ?? existing.state
	};
}

/** Remap subject-owned source devices to the hypothetical perspective node. */
export function remapEdgesForPerspective(
	edges: RenderEdge[],
	perspectiveID: string,
	subjectIDs: Set<string>
): RenderEdge[] {
	const merged = new Map<string, RenderEdge>();

	for (const edge of edges) {
		let from = edge.from;
		let to = edge.to;
		const fromIsSubject = subjectIDs.has(from);
		const toIsSubject = subjectIDs.has(to);

		if (fromIsSubject) from = perspectiveID;
		if (toIsSubject) to = perspectiveID;
		if (from === to) continue;

		const remapped: RenderEdge = {
			...edge,
			id: `${edge.id}:perspective`,
			from,
			to
		};
		const key = mergeKey(remapped);
		const existing = merged.get(key);
		merged.set(key, existing ? mergeEdges(existing, remapped) : remapped);
	}

	return [...merged.values()];
}

/** Subject devices that only appear as collapsed sources (hide from graph). */
export function hiddenSubjectSourceDeviceIds(
	edges: RenderEdge[],
	subjectIDs: Set<string>,
	perspectiveID: string
): Set<string> {
	const destinationSubjects = new Set<string>();
	for (const edge of edges) {
		if (subjectIDs.has(edge.to) && edge.from !== perspectiveID) {
			destinationSubjects.add(edge.to);
		}
	}
	const hidden = new Set<string>();
	for (const id of subjectIDs) {
		if (!destinationSubjects.has(id)) hidden.add(id);
	}
	return hidden;
}
