<script lang="ts">
  import cytoscape, { type Core, type ElementDefinition, type StylesheetJson } from "cytoscape";
  import { onDestroy, onMount } from "svelte";

  import { fetchHealth } from "./lib/api/health";
  import { fetchTailnet } from "./lib/api/tailnet";
  import type { Device, LocalAPIStatusResponse, TopologyResponse } from "./lib/api/schemas";

  let apiStatus = $state("checking");
  let apiVersion = $state("");
  let devices = $state<Device[]>([]);
  let edges = $state<TopologyResponse["edges"]>([]);
  let selectedDevice = $state<Device | undefined>();
  let localApiError = $state<LocalAPIStatusResponse | Error | undefined>();
  let showOffline = $state(true);
  let showSubnetRouters = $state(true);
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

  $effect(() => {
    if (!graphEl || devices.length === 0) {
      return;
    }
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
        edges = value.edges;
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

    const visibleIDs = new Set(visibleDevices.map((device) => device.id));
    const elements: ElementDefinition[] = [
      ...visibleDevices.map((device) => ({
        classes: [
          device.online ? "online" : "offline",
          device.subnetRouter ? "subnet-router" : "",
        ]
          .filter(Boolean)
          .join(" "),
        data: {
          id: device.id,
          label: device.name || device.ip || device.id,
          color: deviceColor(device),
        },
      })),
      ...edges.filter((edge) => visibleIDs.has(edge.from) && visibleIDs.has(edge.to)).map((edge) => ({
        classes: edge.kind,
        data: {
          id: edge.id,
          source: edge.from,
          target: edge.to,
          label: edge.labels?.join(", ") ?? edge.kind,
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
          "border-color": "#36564b",
          "border-width": 2,
          color: "#14251f",
          content: "data(label)",
          "font-size": 12,
          "font-weight": 700,
          height: 42,
          "text-margin-y": -12,
          "text-valign": "top",
          width: 42,
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
        selector: "edge",
        style: {
          "curve-style": "bezier",
          "line-color": "#74857e",
          "target-arrow-color": "#74857e",
          "target-arrow-shape": "triangle",
          label: "data(label)",
          "font-size": 10,
          "text-rotation": "autorotate",
          "text-margin-y": -6,
          width: 2,
        },
      },
      {
        selector: "edge.owner",
        style: {
          "line-color": "#5d7f73",
          "target-arrow-color": "#5d7f73",
        },
      },
      {
        selector: "edge.tag",
        style: {
          "line-color": "#7c6fb0",
          "target-arrow-color": "#7c6fb0",
          "line-style": "dashed",
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
    ];

    graph = cytoscape({
      container: graphEl,
      elements,
      layout: { name: "cose", animate: false, fit: true, padding: 48 },
      style,
    });

    graph.autoungrabify(false);
    graph.on("select", "node", (event) => {
      const id = event.target.id();
      selectedDevice = devices.find((device) => device.id === id);
    });
  }

  function fitGraph() {
    graph?.fit(undefined, 48);
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
                  onclick={() => (selectedDevice = device)}
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
        <div class="graph-controls" aria-label="Graph controls">
          <button type="button" title="Zoom in" onclick={() => zoomGraph(1.2)}>+</button>
          <button type="button" title="Zoom out" onclick={() => zoomGraph(0.8)}>-</button>
          <button type="button" title="Fit to view" onclick={fitGraph}>Fit</button>
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
