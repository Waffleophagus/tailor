# ADR 001: Use Cytoscape.js for Graph Rendering

**Status**: Accepted

**Date**: 2026-05-25

## Context

We need a graph library that can render a Tailscale tailnet topology. A typical tailnet might have 50-500 devices. The graph must support click-to-select interaction (not drag-to-connect) for device metadata viewing and ACL editing.

## Decision

Use **Cytoscape.js** as the sole graph rendering library.

## Alternatives Considered

### Svelte Flow
- **Pros**: Rich built-in drag handles, custom Svelte component nodes, polished node-based UI primitives.
- **Cons**: DOM-based rendering. Performance degrades significantly past 50-100 nodes. Svelte Flow's primary value proposition is drag-to-connect edge editing, which our UX model does not use.
- **Verdict**: Rejected. Overkill for click-to-select and poor for our scale target.

### D3.js (raw)
- **Pros**: Maximum flexibility, force-directed physics, SVG-based.
- **Cons**: We would hand-build every interaction: click events, zoom, pan, node labels, tooltips. Significant boilerplate for a polished UX.
- **Verdict**: Rejected. Too much custom code for a scoped project.

### Cytoscape.js
- **Pros**: Canvas/WebGL rendering (smooth at 500+ nodes), built-in force-directed layouts (`cose`), native click/select events, existing ecosystem.
- **Cons**: Custom in-node UI requires overlay hackery; modal-based editing is required instead of inline editing.
- **Verdict**: Accepted. Canvas performance is the killer feature for instant graph rendering.

## Consequences

- Editing is modal-based: click node → open panel/modal → edit properties. No inline drag-to-edit.
- We must build a custom detail panel outside the canvas for tag/group edits.
- The combination of Cytoscape.js for the graph + Svelte components for the UI chrome is clean and separated.
