<script lang="ts">
  import cytoscape, {
    type Core,
    type ElementDefinition,
    type NodeSingular,
    type StylesheetJson,
  } from "cytoscape";
  import { forceSimulation, forceLink, forceManyBody, forceCenter } from "d3-force";
  import type { SimulationNodeDatum, SimulationLinkDatum } from "d3-force";
  import { onDestroy, onMount } from "svelte";

  import { fetchHealth } from "./lib/api/health";
  import { fetchTailnet } from "./lib/api/tailnet";
  import type { Device, LocalAPIStatusResponse } from "./lib/api/schemas";

  let apiStatus = $state("checking");
  let apiVersion = $state("");
  let devices = $state<Device[]>([]);
  let selectedDevice = $state<Device | undefined>();
  let localApiError = $state<LocalAPIStatusResponse | Error | undefined>();
  let showOffline = $state(true);
  let showSubnetRouters = $state(true);
  let showLabels = $state(false);
  let selectedTag = $state("all");
  let selectedOwner = $state("all");
  let selectedOS = $state("all");
  let colorBy = $state<"status" | "tag" | "owner" | "os">("status");
  let graphEl: HTMLDivElement;
  let graph: Core | undefined;

  const visibleDevices = $derived(
    devices.filter((device) => {
      if (!showOffline && !device.online) {
        return false;
      }
      if (!showSubnetRouters && device.subnetRouter) {
        return false;
      }
      if (selectedTag !== "all" && !device.tags.includes(selectedTag)) {
        return false;
      }
      if (selectedOwner !== "all" && device.owner !== selectedOwner) {
        return false;
      }
      if (selectedOS !== "all" && device.os !== selectedOS) {
        return false;
      }
      return true;
    }),
  );
  const tagOptions = $derived(unique(devices.flatMap((device) => device.tags)));
  const ownerOptions = $derived(unique(devices.map((device) => device.owner).filter(Boolean)));
  const osOptions = $derived(unique(devices.map((device) => device.os).filter(Boolean)));
  const rootDevice = $derived(devices[0]);
  const visibleRootDevice = $derived(
    rootDevice && visibleDevices.some((device) => device.id === rootDevice.id) ? rootDevice : undefined,
  );
  const visibleSpokes = $derived(
    visibleRootDevice
      ? visibleDevices
          .filter((device) => device.id !== visibleRootDevice.id && device.online)
          .map((device) => ({
            id: `local:${visibleRootDevice.id}:${device.id}`,
            from: visibleRootDevice.id,
            to: device.id,
            kind: "local" as const,
          }))
      : [],
  );
  const visibleOnlineCount = $derived(visibleDevices.filter((device) => device.online).length);

  $effect(() => {
    if (!graphEl || devices.length === 0) {
      return;
    }
    void showLabels;
    renderGraph();
  });

  onMount(async () => {
    const health = await fetchHealth();
    health.match({
      ok: (value) => {
        apiStatus = value.status;
        apiVersion = value.version;
      },
      err: (error) => {
        apiStatus = error.message;
      },
    });

    const tailnet = await fetchTailnet();
    tailnet.match({
      ok: (value) => {
        apiStatus = "connected";
        localApiError = undefined;
        devices = value.devices;
        selectedDevice = value.devices[0];
      },
      err: (error) => {
        apiStatus = "LocalAPI unavailable";
        localApiError = error;
      },
    });
  });

  onDestroy(() => {
    graph?.destroy();
  });

  function renderGraph() {
    if (!graphEl) {
      return;
    }

    const positions = graphPositions();
    const elements: ElementDefinition[] = [
      ...visibleDevices.map((device) => ({
        classes: [
          device.online ? "online" : "offline",
          rootDevice?.id === device.id ? "root" : "",
          device.subnetRouter ? "subnet-router" : "",
          showLabels ? "with-labels" : "",
        ]
          .filter(Boolean)
          .join(" "),
        data: {
          id: device.id,
          label: device.name || device.ip || device.id,
          color: deviceColor(device),
          ringColor: rootDevice?.id === device.id ? "#163f31" : device.online ? "#1f7a52" : "#74857e",
        },
        position: positions.get(device.id),
      })),
      ...visibleSpokes.map((edge) => ({
        classes: edge.kind,
        data: {
          id: edge.id,
          source: edge.from,
          target: edge.to,
        },
      })),
    ];

    graph?.destroy();
    const style: StylesheetJson = [
      {
        selector: "node",
        style: {
          "background-color": "data(color)",
          "background-opacity": 1,
          "border-color": "data(ringColor)",
          "border-opacity": 0.82,
          "border-width": 2.5,
          color: "#21352e",
          content: "data(label)",
          "font-size": 12,
          "font-weight": 700,
          height: 42,
          "min-zoomed-font-size": 7,
          "overlay-opacity": 0,
          "text-margin-y": -12,
          "text-outline-color": "#edf4f0",
          "text-outline-width": 2,
          "text-valign": "top",
          "transition-duration": 180,
          "transition-property":
            "background-color, border-color, border-width, height, opacity, underlay-opacity, underlay-padding, width",
          "underlay-color": "#2f9f68",
          "underlay-opacity": 0,
          "underlay-padding": 8,
          "underlay-shape": "ellipse",
          width: 42,
        },
      },
      {
        selector: "node:not(.with-labels)",
        style: {
          content: "",
        },
      },
      {
        selector: "node.online",
        style: {
          "underlay-opacity": 0.08,
        },
      },
      {
        selector: "node.offline",
        style: {
          opacity: 0.68,
        },
      },
      {
        selector: "node.subnet-router",
        style: {
          shape: "round-rectangle",
          width: 56,
        },
      },
      {
        selector: "node.root",
        style: {
          "border-width": 4,
          height: 58,
          "underlay-opacity": 0.2,
          "underlay-padding": 14,
          width: 58,
        },
      },
      {
        selector: "edge",
        style: {
          "curve-style": "bezier",
          "line-color": "#74857e",
          opacity: 0.6,
          "overlay-opacity": 0,
          "transition-duration": 180,
          "transition-property": "line-color, opacity, width",
          width: 1.8,
        },
      },
      {
        selector: "edge.owner",
        style: {
          "line-color": "#5d7f73",
          width: 2.4,
        },
      },
      {
        selector: "edge.tag",
        style: {
          "line-color": "#7c6fb0",
          "target-arrow-color": "#7c6fb0",
          "line-style": "dashed",
          width: 1.7,
        },
      },
      {
        selector: "edge.subnet",
        style: {
          "line-color": "#a5663f",
          "target-arrow-color": "#a5663f",
          "line-style": "dotted",
        },
      },
      {
        selector: "edge.acl",
        style: {
          "line-color": "#438aa1",
          "target-arrow-color": "#438aa1",
          "target-arrow-shape": "triangle",
          width: 2.2,
        },
      },
      {
        selector: "edge.local",
        style: {
          "line-color": "#5d7f73",
          opacity: 0.5,
          width: 1.8,
        },
      },
      {
        selector: ".dim",
        style: {
          opacity: 0.16,
        },
      },
      {
        selector: "edge.focused",
        style: {
          opacity: 0.96,
          width: 3.3,
        },
      },
      {
        selector: "node.focused",
        style: {
          opacity: 1,
          "underlay-opacity": 0.16,
          "underlay-padding": 10,
        },
      },
      {
        selector: "node:selected",
        style: {
          "border-color": "#163f31",
          "border-width": 4,
          height: 54,
          "underlay-opacity": 0.28,
          "underlay-padding": 14,
          width: 54,
        },
      },
    ];

    graph = cytoscape({
      container: graphEl,
      elements,
      layout: graphLayoutOptions(),
      style,
    });

    graph.autoungrabify(false);
    graph.userPanningEnabled(true);
    graph.userZoomingEnabled(true);
    setupPhysics(graph);
    graph.on("select", "node", (event) => {
      const id = event.target.id();
      selectedDevice = devices.find((device) => device.id === id);
      applyGraphFocus(event.target);
      pulseNode(event.target);
    });
    graph.on("mouseover", "node", (event) => {
      applyGraphFocus(event.target);
    });
    graph.on("mouseout", "node", () => {
      const selected = graph?.$("node:selected");
      if (selected?.length) {
        applyGraphFocus(selected[0] as NodeSingular);
        return;
      }
      clearGraphFocus();
    });
    graph.on("tap", (event) => {
      if (event.target === graph) {
        graph?.nodes().unselect();
        clearGraphFocus();
      }
    });
  }

  function fitGraph() {
    graph?.animate({
      fit: { eles: graph.elements(), padding: 64 },
      duration: 260,
      easing: "ease-out-cubic",
    });
  }

  function reflowGraph() {
    graph?.layout(graphLayoutOptions()).run();
  }

  function zoomGraph(delta: number) {
    if (!graph) {
      return;
    }
    graph.zoom({
      level: graph.zoom() * delta,
      renderedPosition: {
        x: graph.width() / 2,
        y: graph.height() / 2,
      },
    });
  }

  function chooseDevice(device: Device) {
    selectedDevice = device;
    const node = graph?.getElementById(device.id);
    if (!node?.length) {
      return;
    }
    graph?.nodes().unselect();
    node.select();
    applyGraphFocus(node);
    graph?.animate({
      center: { eles: node },
      duration: 320,
      easing: "ease-out-cubic",
    });
  }

  function applyGraphFocus(node: NodeSingular) {
    if (!graph) {
      return;
    }
    const neighborhood = node.closedNeighborhood();
    graph.elements().removeClass("dim focused");
    graph.elements().difference(neighborhood).addClass("dim");
    neighborhood.addClass("focused");
  }

  function clearGraphFocus() {
    graph?.elements().removeClass("dim focused");
  }

  function pulseNode(node: NodeSingular) {
    if (prefersReducedMotion()) {
      return;
    }
    node
      .animate({
        style: { "underlay-opacity": 0.42, "underlay-padding": 20 },
        duration: 120,
        easing: "ease-out-cubic",
      })
      .animate({
        style: { "underlay-opacity": 0.28, "underlay-padding": 14 },
        duration: 240,
        easing: "ease-out-cubic",
        complete: () => node.removeStyle("underlay-opacity underlay-padding"),
      });
  }

  function graphLayoutOptions() {
    const animate = !prefersReducedMotion();
    return {
      name: "preset",
      animate,
      animationDuration: animate ? 520 : 0,
      animationEasing: "ease-out-cubic",
      fit: false,
      padding: 56,
    };
  }

  function graphPositions() {
    const width = graphEl?.clientWidth || 900;
    const height = graphEl?.clientHeight || 620;
    const center = { x: width / 2, y: height / 2 };
    const positions = new Map<string, { x: number; y: number }>();

    const rootID = visibleRootDevice?.id;
    const onlinePeers = visibleDevices.filter((device) => device.id !== rootID && device.online);
    const offlinePeers = visibleDevices.filter((device) => device.id !== rootID && !device.online);
    const minDimension = Math.min(width, height);
    const onlineRadius = clamp(onlinePeers.length * 18, 150, Math.max(170, minDimension * 0.34));
    const offlineRadius = clamp(onlineRadius + 92, onlineRadius + 72, Math.max(onlineRadius + 96, minDimension * 0.47));

    if (rootID) {
      positions.set(rootID, center);
    }
    placeOnRing(positions, onlinePeers, center, onlineRadius, -Math.PI / 2);
    placeOnRing(positions, offlinePeers, center, offlineRadius, -Math.PI / 2 + ringOffset(offlinePeers.length));

    return positions;
  }

  interface PhysicsNode extends SimulationNodeDatum {
    id: string;
  }

  function setupPhysics(cy: Core) {
    const whitelistedNodes = new Set<string>();

    cy.edges().forEach((edge) => {
      whitelistedNodes.add(edge.source().id());
      whitelistedNodes.add(edge.target().id());
    });

    const physicsNodesMap = new Map<string, PhysicsNode>();
    cy.nodes().forEach((node) => {
      if (!whitelistedNodes.has(node.id())) return;
      const pos = node.position();
      const pn: PhysicsNode = { id: node.id(), x: pos.x, y: pos.y };
      physicsNodesMap.set(node.id(), pn);
    });

    const nodes = Array.from(physicsNodesMap.values());
    const links: SimulationLinkDatum<PhysicsNode>[] = [];

    cy.edges().forEach((edge) => {
      links.push({
        source: edge.source().id(),
        target: edge.target().id(),
      });
    });

    let sim = forceSimulation(nodes)
      .force(
        "link",
        forceLink(links)
          .id((d) => (d as PhysicsNode).id)
          .distance(180)
          .strength(0.5),
      )
      .force("charge", forceManyBody().strength(-300))
      .alphaDecay(0.02)
      .velocityDecay(0.4);

    let isDragging = false;

    cy.on("grab", "node", () => {
      isDragging = true;
      sim.alpha(0.8).restart();
    });

    cy.on("free", "node", () => {
      isDragging = false;
      sim.alphaDecay(0.1);
    });

    cy.on("drag", "node", (event) => {
      const node = event.target;
      const simNode = physicsNodesMap.get(node.id());
      if (!simNode) return;

      const pos = node.position();
      simNode.x = pos.x;
      simNode.y = pos.y;

      sim.nodes(nodes).alpha(0.8).restart();
    });

    const tickCallback = () => {
      if (!isDragging) return;
      for (const n of nodes) {
        const node = cy.getElementById(n.id);
        if (node.grabbed()) continue;
        if (n.x !== undefined && n.y !== undefined) {
          node.position({ x: n.x, y: n.y });
        }
      }
    };

    sim.on("tick", tickCallback);
  }

  function placeOnRing(
    positions: Map<string, { x: number; y: number }>,
    ringDevices: Device[],
    center: { x: number; y: number },
    radius: number,
    startAngle: number,
  ) {
    ringDevices.forEach((device, index) => {
      const angle = startAngle + (index / ringDevices.length) * Math.PI * 2;
      positions.set(device.id, {
        x: center.x + Math.cos(angle) * radius,
        y: center.y + Math.sin(angle) * radius,
      });
    });
  }

  function ringOffset(count: number) {
    return count > 0 ? Math.PI / count : 0;
  }

  function clamp(value: number, min: number, max: number) {
    return Math.min(Math.max(value, min), max);
  }

  function prefersReducedMotion() {
    return (
      typeof window !== "undefined" &&
      window.matchMedia("(prefers-reduced-motion: reduce)").matches
    );
  }

  function unique(values: string[]) {
    return [...new Set(values)].sort((a, b) => a.localeCompare(b));
  }

  function deviceColor(device: Device) {
    if (colorBy === "status") {
      return device.online ? "#41a86f" : "#9aa7a1";
    }
    const value =
      colorBy === "tag"
        ? (device.tags[0] ?? "untagged")
        : colorBy === "owner"
          ? device.owner
          : device.os;
    return palette(value || "unknown");
  }

  function palette(value: string) {
    const colors = ["#438aa1", "#a5663f", "#7c6fb0", "#b0892f", "#5d7f73", "#b45f74", "#5973b0"];
    let hash = 0;
    for (let i = 0; i < value.length; i += 1) {
      hash = (hash + value.charCodeAt(i) * (i + 1)) % colors.length;
    }
    return colors[hash];
  }

  function localApiErrorMessage(error: LocalAPIStatusResponse | Error | undefined) {
    if (!error) {
      return "";
    }
    if ("available" in error) {
      return error.error ?? `Unable to reach ${error.socketPath}`;
    }
    return error.message;
  }
</script>

<main>
  <section class="workspace">
    <div class="topbar">
      <div>
        <p class="eyebrow">Tailnet topology</p>
        <h1>Tailor</h1>
      </div>
      <div class="status" data-state={apiStatus}>{apiStatus}</div>
    </div>

    <div class="content">
      <aside>
        <h2>Devices</h2>
        <div class="control-stack">
          <label>
            <input bind:checked={showLabels} type="checkbox" />
            Show labels
          </label>
          <label>
            <input bind:checked={showOffline} type="checkbox" />
            Show offline
          </label>
          <label>
            <input bind:checked={showSubnetRouters} type="checkbox" />
            Show subnet routers
          </label>
          <label>
            <span>Tag</span>
            <select bind:value={selectedTag}>
              <option value="all">All tags</option>
              {#each tagOptions as tag}
                <option value={tag}>{tag}</option>
              {/each}
            </select>
          </label>
          <label>
            <span>Owner</span>
            <select bind:value={selectedOwner}>
              <option value="all">All owners</option>
              {#each ownerOptions as owner}
                <option value={owner}>{owner}</option>
              {/each}
            </select>
          </label>
          <label>
            <span>OS</span>
            <select bind:value={selectedOS}>
              <option value="all">All OSes</option>
              {#each osOptions as os}
                <option value={os}>{os}</option>
              {/each}
            </select>
          </label>
          <label>
            <span>Color</span>
            <select bind:value={colorBy}>
              <option value="status">Status</option>
              <option value="tag">Tag</option>
              <option value="owner">Owner</option>
              <option value="os">OS</option>
            </select>
          </label>
        </div>
        {#if devices.length === 0}
          <p>No devices loaded.</p>
        {:else}
          <p class="count">{visibleDevices.length} of {devices.length} devices visible</p>
          <ul class="device-list">
            {#each visibleDevices as device}
              <li>
                <button
                  class:active={selectedDevice?.id === device.id}
                  type="button"
                  onclick={() => chooseDevice(device)}
                >
                  <span class:online={device.online} class="dot"></span>
                  <span>{device.name}</span>
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </aside>

      <section class="graph" aria-label="Topology graph">
        <div class="graph-hud" aria-label="Graph summary">
          <span><strong>{visibleOnlineCount}</strong> online</span>
          <span><strong>{visibleSpokes.length}</strong> links</span>
        </div>
        <div class="graph-controls" aria-label="Graph controls">
          <button type="button" title="Zoom in" onclick={() => zoomGraph(1.2)}>+</button>
          <button type="button" title="Zoom out" onclick={() => zoomGraph(0.8)}>-</button>
          <button type="button" title="Fit to view" onclick={fitGraph}>⌖</button>
          <button type="button" title="Reflow layout" onclick={reflowGraph}>↻</button>
        </div>
        <div bind:this={graphEl} class="graph-canvas"></div>
        {#if localApiError}
          <div class="empty-state">
            <h2>Connect to Tailscale</h2>
            <p>{localApiErrorMessage(localApiError)}</p>
          </div>
        {/if}
      </section>

      <aside>
        <h2>Policy lens</h2>
        <div class="auth-banner">ACL editing requires Phase 2 authentication.</div>
        {#if selectedDevice}
          <dl class="details">
            <div>
              <dt>Name</dt>
              <dd>{selectedDevice.name}</dd>
            </div>
            <div>
              <dt>IP</dt>
              <dd>{selectedDevice.ip || "unknown"}</dd>
            </div>
            <div>
              <dt>Tailscale IPs</dt>
              <dd>{selectedDevice.tailscaleIps.length ? selectedDevice.tailscaleIps.join(", ") : "unknown"}</dd>
            </div>
            <div>
              <dt>Owner</dt>
              <dd>{selectedDevice.owner || "unknown"}</dd>
            </div>
            <div>
              <dt>Tags</dt>
              <dd>{selectedDevice.tags.length ? selectedDevice.tags.join(", ") : "none"}</dd>
            </div>
            <div>
              <dt>Status</dt>
              <dd>{selectedDevice.online ? "online" : "offline"}</dd>
            </div>
            <div>
              <dt>OS</dt>
              <dd>{selectedDevice.os || "unknown"}</dd>
            </div>
            <div>
              <dt>Subnet routes</dt>
              <dd>{selectedDevice.routedSubnets.length ? selectedDevice.routedSubnets.join(", ") : "none"}</dd>
            </div>
          </dl>
        {:else}
          <p>Select a device to inspect tags, owner, and status.</p>
        {/if}
        <p class="meta">API version: {apiVersion || "unknown"}</p>
      </aside>
    </div>
  </section>
</main>
