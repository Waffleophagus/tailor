<script lang="ts">
  let {
    colorBy = $bindable<"status" | "tag" | "owner" | "os">("status"),
    authenticated = false,
    graphMode = $bindable<"focused" | "all">("all"),
    tagOptions = [] as string[],
    ownerOptions = [] as string[],
    osOptions = [] as string[],
  }: {
    colorBy?: "status" | "tag" | "owner" | "os";
    authenticated?: boolean;
    graphMode?: "focused" | "all";
    tagOptions?: string[];
    ownerOptions?: string[];
    osOptions?: string[];
  } = $props();

  const colors = ["#438aa1", "#a5663f", "#7c6fb0", "#b0892f", "#5d7f73", "#b45f74", "#5973b0"];

  function palette(value: string): string {
    let hash = 0;
    for (let i = 0; i < value.length; i += 1) {
      hash = (hash + value.charCodeAt(i) * (i + 1)) % colors.length;
    }
    return colors[hash];
  }

  interface ColorEntry {
    color: string;
    label: string;
  }

  const nodeEntries = $derived.by((): ColorEntry[] => {
    if (colorBy === "status") {
      return [
        { color: "#41a86f", label: "Online" },
        { color: "#9aa7a1", label: "Offline" },
      ];
    }
    const options =
      colorBy === "tag" ? tagOptions : colorBy === "owner" ? ownerOptions : osOptions;
    const maxVisible = 8;
    const visible = options.slice(0, maxVisible);
    return visible.map((value) => ({
      color: palette(value || "unknown"),
      label: value || "unknown",
    }));
  });

  const nodeLegendTitle = $derived.by((): string => {
    if (colorBy === "status") return "Status";
    if (colorBy === "tag") return "Tag";
    if (colorBy === "owner") return "Owner";
    return "OS";
  });

  const lineTitle = $derived.by((): string => {
    if (!authenticated) return "Inferred relationships";
    if (graphMode === "focused") return `ACL focus\u00a0\u2014\u00a0focused`;
    return "ACL access scope";
  });
</script>

<div class="graph-legend" role="region" aria-label="Graph legend">
  <div class="legend-title">{lineTitle}</div>

  {#if !authenticated}
    <div class="legend-section">
      <div class="legend-row">
        <span class="swatch-line" style="border-color: #5d7f73; border-style: solid; border-width: 2.4px;"></span>
        <span>Owner</span>
      </div>
      <div class="legend-row">
        <span class="swatch-line" style="border-color: #7c6fb0; border-style: dashed; border-width: 1.7px;"></span>
        <span>Tag</span>
      </div>
      <div class="legend-row">
        <span class="swatch-line" style="border-color: #a5663f; border-style: dotted; border-width: 1.8px;"></span>
        <span>Subnet</span>
      </div>
    </div>
  {:else}
    <div class="legend-section">
      <div class="legend-row">
        <span class="swatch-line" style="border-color: #438aa1; border-style: solid; border-width: 2.2px;"></span>
        <span>ACL (generic)</span>
      </div>
      <div class="legend-row">
        <span class="swatch-line" style="border-color: #2f9f68; border-style: solid; border-width: 2.8px;"></span>
        <span>SSH (port 22)</span>
      </div>
      <div class="legend-row">
        <span class="swatch-line" style="border-color: #438aa1; border-style: solid; border-width: 2.4px;"></span>
        <span>HTTP/S (80, 443)</span>
      </div>
      <div class="legend-row">
        <span class="swatch-line" style="border-color: #b0892f; border-style: solid; border-width: 3.1px;"></span>
        <span>Broad (all ports)</span>
      </div>
      <div class="legend-row">
        <span class="swatch-line" style="border-color: #7c6fb0; border-style: dashed; border-width: 2.3px;"></span>
        <span>Limited / Custom</span>
      </div>
    </div>
  {/if}

  <div class="legend-divider"></div>

  <div class="legend-section">
    <div class="legend-section-title">{nodeLegendTitle}</div>
    {#each nodeEntries as entry (entry.label)}
      <div class="legend-row">
        <span class="swatch-dot" style="background-color: {entry.color};"></span>
        <span title={entry.label}>{entry.label}</span>
      </div>
    {/each}
  </div>
</div>

<style>
  .graph-legend {
    position: absolute;
    bottom: 0.75rem;
    left: 0.75rem;
    z-index: 10;
    width: 12rem;
    max-height: calc(100% - 1.5rem);
    overflow-y: auto;
    background: oklch(0.985 0.006 158 / 0.95);
    border: 1px solid oklch(0.82 0.018 158);
    border-radius: 8px;
    padding: 0.5rem;
    box-shadow: 0 8px 22px rgb(23 33 38 / 8%);
    font-size: 0.675rem;
    font-weight: 700;
    color: #586761;
    pointer-events: auto;
  }

  .legend-title {
    font-size: 0.6rem;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #8a9590;
    margin-bottom: 0.5rem;
    padding-bottom: 0.25rem;
    border-bottom: 1px solid #e8eeeb;
  }

  .legend-section-title {
    font-size: 0.6rem;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: #8a9590;
    margin-bottom: 0.15rem;
  }

  .legend-section {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }

  .legend-divider {
    height: 1px;
    background-color: #e8eeeb;
    margin: 0.35rem 0;
  }

  .legend-row {
    display: flex;
    align-items: center;
    gap: 0.4rem;
    line-height: 1.2;
    overflow: hidden;
  }

  .legend-row > span:last-child {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }

  .swatch-line {
    display: inline-block;
    width: 1.25rem;
    min-width: 1.25rem;
    height: 0;
    border-top-width: var(--line-width, 2px);
    border-top-style: solid;
    border-radius: 0.0625rem;
    margin-top: 0.0625rem;
  }

  .swatch-dot {
    display: inline-block;
    width: 0.5rem;
    height: 0.5rem;
    border-radius: 50%;
    min-width: 0.5rem;
  }
</style>
