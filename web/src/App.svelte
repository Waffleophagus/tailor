<script lang="ts">
  import cytoscape, { type Core, type ElementDefinition, type StylesheetJson } from "cytoscape";
  import { onDestroy, onMount } from "svelte";

  import { fetchHealth } from "./lib/api/health";
  import { fetchTailnet } from "./lib/api/tailnet";
  import type { Device, LocalAPIStatusResponse, TopologyResponse } from "./lib/api/schemas";

  let apiStatus = $state("checking");
  let apiVersion = $state("");
  let devices = $state<Device[]>([]);
  let selectedDevice = $state<Device | undefined>();
  let localApiError = $state<LocalAPIStatusResponse | Error | undefined>();
  let graphEl: HTMLDivElement;
  let graph: Core | undefined;

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
        renderGraph(value);
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

  function renderGraph(tailnet: TopologyResponse) {
    if (!graphEl) {
      return;
    }

    const elements: ElementDefinition[] = [
      ...tailnet.devices.map((device) => ({
        classes: device.online ? "online" : "offline",
        data: {
          id: device.id,
          label: device.name || device.ip || device.id,
        },
      })),
      ...tailnet.edges.map((edge) => ({
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
          "background-color": "#9aa7a1",
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
        selector: "node.online",
        style: {
          "background-color": "#41a86f",
        },
      },
      {
        selector: "node.offline",
        style: {
          "background-color": "#9aa7a1",
        },
      },
      {
        selector: "edge",
        style: {
          "curve-style": "bezier",
          "line-color": "#74857e",
          "target-arrow-color": "#74857e",
          "target-arrow-shape": "triangle",
          width: 2,
        },
      },
    ];

    graph = cytoscape({
      container: graphEl,
      elements,
      layout: { name: "cose", animate: false, fit: true, padding: 48 },
      style,
    });

    graph.on("select", "node", (event) => {
      const id = event.target.id();
      selectedDevice = devices.find((device) => device.id === id);
    });
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
        {#if devices.length === 0}
          <p>No devices loaded.</p>
        {:else}
          <ul class="device-list">
            {#each devices as device}
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
              <dt>Owner</dt>
              <dd>{selectedDevice.owner || "unknown"}</dd>
            </div>
            <div>
              <dt>Tags</dt>
              <dd>{selectedDevice.tags.length ? selectedDevice.tags.join(", ") : "none"}</dd>
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
