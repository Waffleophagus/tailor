<script lang="ts">
  import cytoscape, {
    type Core,
    type ElementDefinition,
    type EdgeSingular,
    type NodeSingular,
    type StylesheetJson,
  } from "cytoscape";
  import { forceSimulation, forceLink, forceManyBody } from "d3-force";
  import type { SimulationNodeDatum, SimulationLinkDatum } from "d3-force";
  import { onDestroy, onMount } from "svelte";

  import {
    authenticateCloud,
    draftPolicyRule,
    fetchCloudStatus,
    fetchPolicy,
    saveValidatedPolicyDraft,
    validatePolicyDraft,
  } from "./lib/api/cloud";
  import { fetchHealth } from "./lib/api/health";
  import type { CloudAuthStatusResponse, Device, Edge, LocalAPIStatusResponse, PolicyResponse } from "./lib/api/schemas";
  import { fetchTopology } from "./lib/api/topology";
  import { connectTopologySocket } from "./lib/api/topologySocket";
  import SidebarLeft from "./lib/components/SidebarLeft.svelte";
  import SidebarRight from "./lib/components/SidebarRight.svelte";
  import SidebarToggleButton from "./lib/components/SidebarToggleButton.svelte";
  import GraphLegend from "./lib/components/GraphLegend.svelte";

  let apiStatus = $state("checking");
  let apiVersion = $state("");
  let devices = $state<Device[]>([]);
  let edges = $state<Edge[]>([]);
  let selectedDevice = $state<Device | undefined>();
  let localApiError = $state<LocalAPIStatusResponse | Error | undefined>();
  let cloudStatus = $state<CloudAuthStatusResponse>({ authenticated: false, hasPolicy: false });
  let policy = $state<PolicyResponse | undefined>();
  let draftHuJSON = $state("");
  let draftRuleText = $state("");
  let editSource = $state("");
  let editDestination = $state("");
  let editPortPreset = $state("443");
  let editCustomPorts = $state("");
  let editStatus = $state("");
  let editBusy = $state(false);
  let draftValid = $state(false);
  let phase2Open = $state(false);
  let policyOpen = $state(false);
  let cloudError = $state("");
  let cloudBusy = $state(false);
  let authTailnet = $state("-");
  let authAPIKey = $state("");
  let showOffline = $state(true);
  let showSubnetRouters = $state(true);
  let showTailnet = $state(false);
  let showLabels = $state(false);
  let graphMode = $state<"focused" | "all">("focused");
  let selectedTag = $state("all");
  let selectedOwner = $state("all");
  let selectedOS = $state("all");
  let colorBy = $state<"status" | "tag" | "owner" | "os">("status");
  let graphEl: HTMLDivElement;
  let graph: Core | undefined;
  let cleanupPhysics: (() => void) | undefined;
  let physicsSignature = "";
  let layoutSignature = "";
  let graphDragActive = false;
  let graphDragMoved = false;
  let lastGraphDragEndAt = 0;
  let pendingLayoutSync = false;
  let hoveredNodeID: string | undefined;
  let disconnectTopologySocket: (() => void) | undefined;
  let leftOpen = $state(true);
  let rightOpen = $state(true);
  const deviceAngles = new Map<string, number>();
  const lastOnlineState = new Map<string, boolean>();

  interface RenderEdge {
    id: string;
    from: string;
    to: string;
    kind: string;
    accessScope?: Edge["accessScope"];
  }

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
  const graphRootDevice = $derived(cloudStatus.authenticated && graphMode === "focused" ? (selectedDevice ?? rootDevice) : rootDevice);
  const visibleDeviceIDs = $derived(new Set(visibleDevices.map((device) => device.id)));
  const visibleEdges = $derived(graphEdges());
  const graphDevices = $derived(devicesForGraph());
  const visibleOnlineCount = $derived(visibleDevices.filter((device) => device.online).length);
  const graphOnlineCount = $derived(graphDevices.filter((device) => device.online).length);

  $effect(() => {
    if (!graphEl) {
      return;
    }
    if (devices.length === 0) {
      cleanupPhysics?.();
      cleanupPhysics = undefined;
      physicsSignature = "";
      layoutSignature = "";
      graphDragActive = false;
      graphDragMoved = false;
      lastGraphDragEndAt = 0;
      pendingLayoutSync = false;
      hoveredNodeID = undefined;
      graph?.destroy();
      graph = undefined;
      return;
    }
    void showLabels;
    void edges;
    void graphMode;
    void selectedDevice;
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

    const cloud = await fetchCloudStatus();
    cloud.match({
      ok: (value) => {
        cloudStatus = value;
      },
      err: (error) => {
        cloudError = error.message;
      },
    });

    disconnectTopologySocket = connectTopologySocket({
      onSnapshot: (value) => {
        apiStatus = "connected";
        localApiError = undefined;
        devices = value.devices;
        edges = value.edges;
        selectedDevice = selectedDevice
          ? (value.devices.find((device) => device.id === selectedDevice?.id) ?? value.devices[0])
          : value.devices[0];
      },
      onUnavailable: (status) => {
        apiStatus = "LocalAPI unavailable";
        localApiError = status;
        devices = [];
        edges = [];
        selectedDevice = undefined;
      },
      onConnectionState: (state) => {
        if (state === "connected" && devices.length > 0) {
          apiStatus = "connected";
          return;
        }
        apiStatus = state;
      },
      onError: (error) => {
        if (devices.length === 0) {
          apiStatus = "socket error";
          localApiError = error;
        }
      },
    });
  });

  onDestroy(() => {
    disconnectTopologySocket?.();
    cleanupPhysics?.();
    graph?.destroy();
  });

  function renderGraph() {
    if (!graphEl) {
      return;
    }

    if (graph) {
      syncGraph();
      return;
    }

    const elements = graphElements(graphPositions());
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
        selector: "node.entering",
        style: {
          opacity: 0,
          "underlay-opacity": 0.26,
          "underlay-padding": 18,
        },
      },
      {
        selector: "node.hide-labels",
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
        selector: "edge.scope-ssh",
        style: {
          "line-color": "#2f9f68",
          "target-arrow-color": "#2f9f68",
          width: 2.8,
        },
      },
      {
        selector: "edge.scope-http",
        style: {
          "line-color": "#438aa1",
          "target-arrow-color": "#438aa1",
          width: 2.4,
        },
      },
      {
        selector: "edge.scope-broad",
        style: {
          "line-color": "#b0892f",
          "target-arrow-color": "#b0892f",
          width: 3.1,
        },
      },
      {
        selector: "edge.scope-custom, edge.scope-limited",
        style: {
          "line-color": "#7c6fb0",
          "target-arrow-color": "#7c6fb0",
          "line-style": "dashed",
          width: 2.3,
        },
      },
      {
        selector: "edge.local",
        style: {
          "curve-style": "straight",
          "line-color": "#2f9f68",
          opacity: 0.66,
          width: 2.2,
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
        selector: "node.selected",
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
      boxSelectionEnabled: false,
      selectionType: "single",
      style,
    });
    layoutSignature = graphLayoutSignature();

    graph.autoungrabify(false);
    graph.autounselectify(true);
    graph.userPanningEnabled(true);
    graph.userZoomingEnabled(true);
    refreshPhysics(graph);
    graph.on("tap", "node", (event) => {
      if (graphDragMoved || Date.now() - lastGraphDragEndAt < 120) {
        return;
      }
      selectGraphNode(event.target as NodeSingular, { pulse: true });
    });
    graph.on("mouseover", "node", (event) => {
      hoveredNodeID = event.target.id();
      applyGraphFocus(event.target as NodeSingular);
    });
    graph.on("mouseout", "node", (event) => {
      if (hoveredNodeID === event.target.id()) {
        hoveredNodeID = undefined;
      }
      applyCurrentGraphFocus();
    });
    graph.on("tap", (event) => {
      if (event.target === graph) {
        hoveredNodeID = undefined;
        clearGraphFocus();
      }
    });
    updateGraphSelection();
    applyCurrentGraphFocus();
    rememberOnlineState();
    window.requestAnimationFrame(() => fitGraph());
  }

  function fitGraph() {
    graph?.animate({
      fit: { eles: graph.elements(), padding: 64 },
      duration: 260,
      easing: "ease-out-cubic",
    });
  }

  function reflowGraph() {
    animateGraphToPositions(graphPositions(), 420);
    layoutSignature = graphLayoutSignature();
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
    updateGraphSelection();
    window.requestAnimationFrame(() => {
      const node = graph?.getElementById(device.id);
      if (!node?.length) {
        return;
      }
      updateGraphSelection();
      applyGraphFocus(node);
      graph?.animate({
        center: { eles: node },
        duration: 320,
        easing: "ease-out-cubic",
      });
    });
  }

  async function enableACLEditing() {
    if (cloudBusy) {
      return;
    }
    cloudBusy = true;
    cloudError = "";
    const result = await authenticateCloud({
      tailnet: authTailnet.trim() || "-",
      apiKey: authAPIKey,
    });
    await result.match({
      ok: async (value) => {
        const topology = await fetchTopology();
        if (topology.isErr()) {
          cloudError = topology.error.message;
          return;
        }
        devices = topology.value.devices;
        edges = topology.value.edges;
        selectedDevice = selectedDevice
          ? (topology.value.devices.find((device) => device.id === selectedDevice?.id) ?? topology.value.devices[0])
          : topology.value.devices[0];
        cloudStatus = value;
        authAPIKey = "";
        phase2Open = false;
        await loadPolicy();
      },
      err: async (error) => {
        cloudError = error.message;
      },
    });
    cloudBusy = false;
  }

  async function loadPolicy() {
    const result = await fetchPolicy();
    result.match({
      ok: (value) => {
        policy = value;
        draftHuJSON = "";
        draftRuleText = "";
        draftValid = false;
        policyOpen = true;
        cloudError = "";
      },
      err: (error) => {
        cloudError = error.message;
      },
    });
  }

  async function createPolicyDraft() {
    if (editBusy) {
      return;
    }
    editBusy = true;
    cloudError = "";
    editStatus = "";
    draftValid = false;
    const result = await draftPolicyRule({
      sources: splitSelectors(editSource),
      destinations: splitSelectors(editDestination),
      ports: selectedPorts(),
      protocol: "tcp",
    });
    result.match({
      ok: (value) => {
        draftHuJSON = value.hujson;
        draftRuleText = JSON.stringify(value.rule, null, 2);
        editStatus = "Draft ready. Validate before saving.";
      },
      err: (error) => {
        cloudError = error.message;
      },
    });
    editBusy = false;
  }

  async function validateDraft() {
    if (editBusy || !draftHuJSON) {
      return;
    }
    editBusy = true;
    cloudError = "";
    const result = await validatePolicyDraft(draftHuJSON);
    result.match({
      ok: (value) => {
        draftValid = value.valid;
        editStatus = value.valid ? "Draft validated. Save is enabled." : (value.errors ?? ["Draft failed validation."]).join(" ");
      },
      err: (error) => {
        draftValid = false;
        cloudError = error.message;
      },
    });
    editBusy = false;
  }

  async function saveDraft() {
    if (editBusy || !draftValid) {
      return;
    }
    editBusy = true;
    cloudError = "";
    const result = await saveValidatedPolicyDraft();
    result.match({
      ok: (value) => {
        policy = { tailnet: value.tailnet, hujson: value.hujson };
        draftHuJSON = "";
        draftRuleText = "";
        draftValid = false;
        editStatus = "Saved. Topology will refresh from the updated policy.";
      },
      err: (error) => {
        cloudError = error.message;
      },
    });
    editBusy = false;
  }

  function closePhase2Dialog() {
    if (cloudBusy) {
      return;
    }
    phase2Open = false;
  }

  function selectGraphNode(node: NodeSingular, options: { pulse?: boolean } = {}) {
    const device = devices.find((candidate) => candidate.id === node.id());
    if (!device) {
      return;
    }
    selectedDevice = device;
    updateGraphSelection();
    applyGraphFocus(node);
    if (options.pulse) {
      pulseNode(node);
    }
  }

  function updateGraphSelection() {
    if (!graph) {
      return;
    }
    const selectedID = selectedDevice?.id;
    graph.nodes().forEach((node) => {
      node.toggleClass("selected", selectedID === node.id());
    });
  }

  function applyCurrentGraphFocus() {
    if (!graph) {
      return;
    }
    const focusID = hoveredNodeID ?? selectedDevice?.id;
    if (!focusID) {
      clearGraphFocus();
      return;
    }
    const node = graph.getElementById(focusID);
    if (!node.length) {
      clearGraphFocus();
      return;
    }
    applyGraphFocus(node);
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

  function graphElements(positions: Map<string, { x: number; y: number }>) {
    const elements: ElementDefinition[] = [
      ...graphDevices.map((device) => ({
        classes: deviceClasses(device),
        data: deviceData(device),
        position: positions.get(device.id),
      })),
      ...visibleEdges.map((edge) => ({
        classes: edgeClasses(edge),
        data: {
          id: edge.id,
          source: edge.from,
          target: edge.to,
        },
      })),
    ];
    return elements;
  }

  function syncGraph() {
    if (!graph) {
      return;
    }

    const desiredNodeIDs = new Set(graphDevices.map((device) => device.id));
    const desiredEdgeIDs = new Set(visibleEdges.map((edge) => edge.id));
    const positions = graphPositions();
    const nextLayoutSignature = graphLayoutSignature();
    const layoutChanged = nextLayoutSignature !== layoutSignature;
    if (layoutChanged && graphDragActive) {
      pendingLayoutSync = true;
    }

    graph.edges().forEach((edge) => {
      if (!desiredEdgeIDs.has(edge.id())) {
        removeEdge(edge);
      }
    });
    graph.nodes().forEach((node) => {
      if (!desiredNodeIDs.has(node.id())) {
        removeNode(node);
      }
    });

    for (const device of graphDevices) {
      const targetPosition = positions.get(device.id);
      if (!targetPosition) {
        continue;
      }
      const node = graph.getElementById(device.id);
      const wasOnline = lastOnlineState.get(device.id);
      if (!node.length) {
        addNode(device, targetPosition);
        continue;
      }
      node.data(deviceData(device));
      node.classes(deviceClasses(device));
      if (layoutChanged && !graphDragActive && !node.grabbed()) {
        animateNodeTo(node, targetPosition, wasOnline === false && device.online);
      }
    }

    for (const edge of visibleEdges) {
      const existing = graph.getElementById(edge.id);
      if (existing.length) {
        existing.classes(edgeClasses(edge));
        continue;
      }
      addEdge(edge);
    }

    if (!graphDragActive) {
      layoutSignature = nextLayoutSignature;
    }
    rememberOnlineState();
    refreshPhysics(graph);
    updateGraphSelection();
    applyCurrentGraphFocus();
  }

  function addNode(device: Device, targetPosition: { x: number; y: number }) {
    if (!graph) {
      return;
    }
    const rootPosition = graphRootDevice ? graph.getElementById(graphRootDevice.id).position() : undefined;
    const startPosition =
      device.online && graphRootDevice?.id !== device.id && rootPosition?.x !== undefined
        ? rootPosition
        : targetPosition;
    const node = graph.add({
      group: "nodes",
      classes: `${deviceClasses(device)} entering`,
      data: deviceData(device),
      position: startPosition,
    });
    if (prefersReducedMotion() || graphDragActive) {
      node.removeClass("entering");
      node.position(targetPosition);
      return;
    }
    node.animate(
      {
        position: targetPosition,
        style: { opacity: device.online ? 1 : 0.68, "underlay-padding": 8 },
      },
      {
        duration: 360,
        easing: "ease-out-cubic",
        complete: () => {
          node.removeClass("entering");
          node.removeStyle("opacity underlay-padding");
        },
      },
    );
  }

  function addEdge(edge: RenderEdge) {
    if (!graph) {
      return;
    }
    const added = graph.add({
      group: "edges",
      classes: edgeClasses(edge),
      data: {
        id: edge.id,
        source: edge.from,
        target: edge.to,
      },
    });
    if (prefersReducedMotion()) {
      return;
    }
    added.style({ opacity: 0, width: 0.2 });
    added.animate(
      { style: { opacity: 0.66, width: 2.2 } },
      {
        duration: 260,
        easing: "ease-out-cubic",
        complete: () => added.removeStyle("opacity width"),
      },
    );
  }

  function removeNode(node: NodeSingular) {
    if (prefersReducedMotion()) {
      node.remove();
      return;
    }
    node.animate(
      { style: { opacity: 0, "underlay-opacity": 0 } },
      {
        duration: 180,
        easing: "ease-out-cubic",
        complete: () => node.remove(),
      },
    );
  }

  function removeEdge(edge: EdgeSingular) {
    if (prefersReducedMotion()) {
      edge.remove();
      return;
    }
    edge.animate(
      { style: { opacity: 0, width: 0.2 } },
      {
        duration: 160,
        easing: "ease-out-cubic",
        complete: () => edge.remove(),
      },
    );
  }

  function animateNodeTo(
    node: NodeSingular,
    targetPosition: { x: number; y: number },
    becameOnline: boolean,
  ) {
    const currentPosition = node.position();
    const moved =
      Math.abs(currentPosition.x - targetPosition.x) > 1 ||
      Math.abs(currentPosition.y - targetPosition.y) > 1;
    node.stop(true, false);
    if (moved && !prefersReducedMotion()) {
      node.animate(
        { position: targetPosition },
        { duration: becameOnline ? 420 : 280, easing: "ease-out-cubic" },
      );
    } else {
      node.position(targetPosition);
    }
    if (becameOnline) {
      pulseNode(node);
    }
  }

  function animateGraphToPositions(positions: Map<string, { x: number; y: number }>, duration: number) {
    graph?.nodes().forEach((node) => {
      const position = positions.get(node.id());
      if (!position) {
        return;
      }
      node.stop(true, false);
      if (prefersReducedMotion()) {
        node.position(position);
        return;
      }
      node.animate({ position }, { duration, easing: "ease-out-cubic" });
    });
  }

  function spreadGraphAfterDrag(draggedID: string, onComplete?: () => void) {
    if (!graph) {
      onComplete?.();
      return;
    }

    const positions = graphSpreadPositionsAfterDrag(draggedID);
    let pendingAnimations = 0;
    const finishAnimation = () => {
      pendingAnimations -= 1;
      if (pendingAnimations === 0) {
        onComplete?.();
      }
    };

    graph.nodes().forEach((node) => {
      if (node.id() === draggedID) {
        return;
      }
      const position = positions.get(node.id());
      if (!position) {
        return;
      }
      node.stop(true, false);
      if (prefersReducedMotion()) {
        node.position(position);
        return;
      }
      pendingAnimations += 1;
      node.animate(
        { position },
        {
          duration: 380,
          easing: "ease-out-cubic",
          complete: finishAnimation,
        },
      );
    });

    layoutSignature = graphLayoutSignature();
    if (pendingAnimations === 0) {
      onComplete?.();
    }
  }

  function deviceClasses(device: Device) {
    return [
      device.online ? "online" : "offline",
      graphRootDevice?.id === device.id ? "root" : "",
      selectedDevice?.id === device.id ? "selected" : "",
      device.subnetRouter ? "subnet-router" : "",
      showLabels ? "with-labels" : "hide-labels",
    ]
      .filter(Boolean)
      .join(" ");
  }

  function deviceData(device: Device) {
    return {
      id: device.id,
      label: device.name || device.ip || device.id,
      color: deviceColor(device),
      ringColor: graphRootDevice?.id === device.id ? "#163f31" : device.online ? "#1f7a52" : "#74857e",
    };
  }

  function graphEdges(): RenderEdge[] {
    if (cloudStatus.authenticated && edges.length > 0) {
      const renderedEdges = edges
        .filter((edge) => visibleDeviceIDs.has(edge.from) && visibleDeviceIDs.has(edge.to))
        .map((edge) => ({
          id: edge.id,
          from: edge.from,
          to: edge.to,
          kind: edge.kind,
          accessScope: edge.accessScope,
        }));
      if (graphMode === "all") {
        return renderedEdges;
      }
      const focusID = graphRootDevice?.id;
      if (!focusID) {
        return [];
      }
      return renderedEdges.filter((edge) => edge.from === focusID || edge.to === focusID);
    }
    const root = rootDevice;
    if (!root || !visibleDeviceIDs.has(root.id) || !root.online) {
      return [];
    }

    return visibleDevices
      .filter((device) => device.id !== root.id && device.online)
      .map((device) => ({
        id: `local:${root.id}:${device.id}`,
        from: root.id,
        to: device.id,
        kind: "local",
      }));
  }

  function devicesForGraph() {
    if (!cloudStatus.authenticated || graphMode === "all" || edges.length === 0) {
      return visibleDevices;
    }
    const ids = new Set<string>();
    if (graphRootDevice?.id && visibleDeviceIDs.has(graphRootDevice.id)) {
      ids.add(graphRootDevice.id);
    }
    for (const edge of visibleEdges) {
      ids.add(edge.from);
      ids.add(edge.to);
    }
    return visibleDevices.filter((device) => ids.has(device.id));
  }

  function edgeClasses(edge: RenderEdge) {
    return [edge.kind, edge.accessScope ? `scope-${edge.accessScope}` : ""].filter(Boolean).join(" ");
  }

  function splitSelectors(value: string) {
    return value
      .split(",")
      .map((part) => part.trim())
      .filter(Boolean);
  }

  function selectedPorts() {
    if (editPortPreset === "custom") {
      return splitSelectors(editCustomPorts);
    }
    return splitSelectors(editPortPreset);
  }

  function graphPositions() {
    const width = graphEl?.clientWidth || 900;
    const height = graphEl?.clientHeight || 620;
    const center = { x: width / 2, y: height / 2 };
    const positions = new Map<string, { x: number; y: number }>();

    const rootID = graphRootDevice && graphDevices.some((device) => device.id === graphRootDevice.id) ? graphRootDevice.id : undefined;
    const onlinePeers = graphDevices.filter((device) => device.id !== rootID && device.online);
    const offlinePeers = graphDevices.filter((device) => device.id !== rootID && !device.online);
    const minDimension = Math.min(width, height);
    const onlineRadius = clamp(onlinePeers.length * 18, 150, Math.max(170, minDimension * 0.34));
    const offlineRadius = clamp(onlineRadius + 92, onlineRadius + 72, Math.max(onlineRadius + 96, minDimension * 0.47));

    if (rootID) {
      positions.set(rootID, center);
    }
    placeOnRing(positions, onlinePeers, center, onlineRadius);
    placeOnRing(positions, offlinePeers, center, offlineRadius);

    return positions;
  }

  function graphSpreadPositionsAfterDrag(draggedID: string) {
    const positions = graphPositions();
    const rootID = graphRootDevice && graphDevices.some((device) => device.id === graphRootDevice.id) ? graphRootDevice.id : undefined;
    if (draggedID === rootID && rootID) {
      const droppedRootPosition = graph?.getElementById(rootID).position();
      const plannedRootPosition = positions.get(rootID);
      if (droppedRootPosition && plannedRootPosition) {
        const offset = {
          x: droppedRootPosition.x - plannedRootPosition.x,
          y: droppedRootPosition.y - plannedRootPosition.y,
        };
        positions.forEach((position, id) => {
          if (id !== draggedID) {
            positions.set(id, {
              x: position.x + offset.x,
              y: position.y + offset.y,
            });
          }
        });
      }
    }

    const draggedPosition = graph?.getElementById(draggedID).position();
    if (!draggedPosition) {
      return positions;
    }
    positions.forEach((position, id) => {
      if (id !== draggedID) {
        positions.set(id, separatePosition(position, draggedPosition, minNodeSpacing(id, draggedID)));
      }
    });
    return positions;
  }

  function separatePosition(
    position: { x: number; y: number },
    avoidedPosition: { x: number; y: number },
    minDistance: number,
  ) {
    const dx = position.x - avoidedPosition.x;
    const dy = position.y - avoidedPosition.y;
    const distance = Math.hypot(dx, dy);
    if (distance >= minDistance) {
      return position;
    }
    const angle = distance > 0.1 ? Math.atan2(dy, dx) : -Math.PI / 2;
    return {
      x: avoidedPosition.x + Math.cos(angle) * minDistance,
      y: avoidedPosition.y + Math.sin(angle) * minDistance,
    };
  }

  function minNodeSpacing(nodeID: string, avoidedNodeID: string) {
    const node = graph?.getElementById(nodeID);
    const avoidedNode = graph?.getElementById(avoidedNodeID);
    return nodeRadius(node) + nodeRadius(avoidedNode) + 24;
  }

  function nodeRadius(node: NodeSingular | undefined) {
    if (!node?.length) {
      return 28;
    }
    const width = Number(node.style("width"));
    const height = Number(node.style("height"));
    return Math.max(Number.isFinite(width) ? width : 56, Number.isFinite(height) ? height : 56) / 2;
  }

  function graphLayoutSignature() {
    return [
      graphRootDevice?.id ?? "",
      graphDevices.map((device) => device.id).sort().join(","),
      visibleEdges.map((edge) => edge.id).sort().join(","),
      graphEl?.clientWidth ?? 0,
      graphEl?.clientHeight ?? 0,
    ].join("|");
  }

  interface PhysicsNode extends SimulationNodeDatum {
    id: string;
  }

  function refreshPhysics(cy: Core) {
    const signature = [
      cy.nodes().map((node) => node.id()).sort().join(","),
      cy.edges().map((edge) => edge.id()).sort().join(","),
    ].join("|");
    if (signature === physicsSignature) {
      return;
    }
    cleanupPhysics?.();
    cleanupPhysics = setupPhysics(cy);
    physicsSignature = signature;
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

    const sim = forceSimulation(nodes)
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

    const syncSimulationNodesFromGraph = () => {
      for (const [id, simNode] of physicsNodesMap) {
        const node = cy.getElementById(id);
        if (!node.length) {
          continue;
        }
        const pos = node.position();
        simNode.x = pos.x;
        simNode.y = pos.y;
        simNode.vx = 0;
        simNode.vy = 0;
      }
    };

    const onGrab = (event: cytoscape.EventObject) => {
      const node = event.target as NodeSingular;
      const simNode = physicsNodesMap.get(node.id());
      cy.elements().stop(true, false);
      graphDragActive = true;
      graphDragMoved = false;
      syncSimulationNodesFromGraph();
      if (simNode) {
        const pos = node.position();
        simNode.fx = pos.x;
        simNode.fy = pos.y;
      }
      isDragging = true;
      sim.alphaDecay(0.02);
      sim.alpha(0.8).restart();
    };

    const onFree = (event: cytoscape.EventObject) => {
      const node = event.target as NodeSingular;
      const simNode = physicsNodesMap.get(node.id());
      const didMove = graphDragMoved;
      if (simNode) {
        const pos = node.position();
        simNode.x = pos.x;
        simNode.y = pos.y;
        simNode.fx = undefined;
        simNode.fy = undefined;
      }
      isDragging = false;
      graphDragActive = false;
      if (didMove) {
        lastGraphDragEndAt = Date.now();
      }
      sim.alphaDecay(0.1);
      sim.stop();
      window.setTimeout(() => {
        graphDragMoved = false;
      }, 120);
      if (pendingLayoutSync) {
        pendingLayoutSync = false;
        syncGraph();
      } else if (didMove) {
        spreadGraphAfterDrag(node.id(), syncSimulationNodesFromGraph);
      }
    };

    const onDrag = (event: cytoscape.EventObject) => {
      const node = event.target as NodeSingular;
      const simNode = physicsNodesMap.get(node.id());
      if (!simNode) return;

      graphDragMoved = true;
      const pos = node.position();
      simNode.x = pos.x;
      simNode.y = pos.y;
      simNode.fx = pos.x;
      simNode.fy = pos.y;

      sim.nodes(nodes).alpha(0.8).restart();
    };

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
    cy.on("grab", "node", onGrab);
    cy.on("free", "node", onFree);
    cy.on("drag", "node", onDrag);

    return () => {
      cy.off("grab", "node", onGrab);
      cy.off("free", "node", onFree);
      cy.off("drag", "node", onDrag);
      graphDragActive = false;
      sim.stop();
    };
  }

  function placeOnRing(
    positions: Map<string, { x: number; y: number }>,
    ringDevices: Device[],
    center: { x: number; y: number },
    radius: number,
  ) {
    ringDevices.forEach((device) => {
      const angle = wheelAngle(device.id);
      positions.set(device.id, {
        x: center.x + Math.cos(angle) * radius,
        y: center.y + Math.sin(angle) * radius,
      });
    });
  }

  function wheelAngle(id: string) {
    const existing = deviceAngles.get(id);
    if (existing !== undefined) {
      return existing;
    }

    const angles = Array.from(deviceAngles.values())
      .map(normalizeAngle)
      .sort((a, b) => a - b);
    let angle = -Math.PI / 2;
    if (angles.length === 1) {
      angle = angles[0] + Math.PI;
    } else if (angles.length > 1) {
      let bestStart = angles[angles.length - 1];
      let bestGap = angles[0] + Math.PI * 2 - bestStart;
      for (let i = 0; i < angles.length - 1; i += 1) {
        const gap = angles[i + 1] - angles[i];
        if (gap > bestGap) {
          bestGap = gap;
          bestStart = angles[i];
        }
      }
      angle = bestStart + bestGap / 2;
    }
    deviceAngles.set(id, angle);
    return angle;
  }

  function normalizeAngle(angle: number) {
    const fullCircle = Math.PI * 2;
    return ((angle % fullCircle) + fullCircle) % fullCircle;
  }

  function rememberOnlineState() {
    for (const device of devices) {
      lastOnlineState.set(device.id, device.online);
    }
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
      return error.error ?? `Unable to reach ${error.localApiEndpoint}`;
    }
    return error.message;
  }

  function phaseLabel() {
    return cloudStatus.authenticated ? "Phase 2" : "Phase 1";
  }
</script>

<main>
  <section class="workspace">
    <div class="topbar">
      <div>
        <p class="eyebrow">Tailnet topology</p>
        <h1>Tailor</h1>
      </div>
      <div class="topbar-actions">
        <div class="phase-chip" data-authenticated={cloudStatus.authenticated}>
          <span>{phaseLabel()}</span>
          {#if cloudStatus.authenticated}
            <strong>{cloudStatus.tailnet}</strong>
          {:else}
            <strong>LocalAPI only</strong>
          {/if}
        </div>
        {#if cloudStatus.authenticated}
          <div class="graph-mode-toggle" aria-label="Policy Lens graph mode">
            <button
              type="button"
              class:active={graphMode === "focused"}
              onclick={() => (graphMode = "focused")}
            >
              Focused
            </button>
            <button
              type="button"
              class:active={graphMode === "all"}
              onclick={() => (graphMode = "all")}
            >
              All connections
            </button>
          </div>
        {/if}
        {#if cloudStatus.authenticated}
          <button class="secondary-btn" type="button" onclick={loadPolicy}>Raw HuJSON</button>
        {:else}
          <button class="primary-btn" type="button" onclick={() => (phase2Open = true)}>Enable ACL Editing</button>
        {/if}
        <div class="status" data-state={apiStatus}><span class="status-pip"></span>{apiStatus}</div>
      </div>
    </div>

    <div class="content">
      <SidebarLeft
        bind:open={leftOpen}
        {devices}
        {visibleDevices}
        bind:selectedDevice
        bind:showLabels
        bind:showOffline
        bind:showSubnetRouters
        bind:showTailnet
        bind:selectedTag
        bind:selectedOwner
        bind:selectedOS
        bind:colorBy
        {tagOptions}
        {ownerOptions}
        {osOptions}
        {visibleOnlineCount}
        {chooseDevice}
      />

      <section class="graph" aria-label="Topology graph">
        <SidebarToggleButton
          position="left"
          open={leftOpen}
          ontoggle={() => (leftOpen = !leftOpen)}
        />
        <SidebarToggleButton
          position="right"
          open={rightOpen}
          ontoggle={() => (rightOpen = !rightOpen)}
        />

        <div class="graph-hud" aria-label="Graph summary">
          <span><strong>{graphOnlineCount}</strong> online</span>
          <span><strong>{visibleEdges.length}</strong> links</span>
          {#if cloudStatus.authenticated && graphMode === "focused" && graphRootDevice}
            <span><strong>{graphRootDevice.name || graphRootDevice.ip}</strong> focus</span>
          {/if}
        </div>
        <div class="graph-controls" aria-label="Graph controls">
          <button type="button" title="Zoom in" onclick={() => zoomGraph(1.2)}>+</button>
          <button type="button" title="Zoom out" onclick={() => zoomGraph(0.8)}>-</button>
          <button type="button" title="Fit to view" onclick={fitGraph}>⌖</button>
          <button type="button" title="Reflow layout" onclick={reflowGraph}>↻</button>
        </div>
        <div bind:this={graphEl} class="graph-canvas"></div>
        <GraphLegend {colorBy} authenticated={cloudStatus.authenticated} {graphMode} {tagOptions} {ownerOptions} {osOptions} />
        {#if localApiError}
          <div class="empty-state">
            <h2>Connect to Tailscale</h2>
            <p>{localApiErrorMessage(localApiError)}</p>
          </div>
        {/if}
        {#if policyOpen && policy}
          <section class="policy-panel" aria-label="Raw HuJSON policy">
            <div class="policy-panel-header">
              <div>
                <p class="eyebrow">Policy editor</p>
                <h2>{policy.tailnet}</h2>
              </div>
              <button type="button" title="Close policy panel" onclick={() => (policyOpen = false)}>×</button>
            </div>
            <div class="policy-editor">
              <form class="acl-edit-form" onsubmit={(event) => { event.preventDefault(); void createPolicyDraft(); }}>
                <label>
                  <span>Sources</span>
                  <input bind:value={editSource} placeholder="alice@example.com, group:eng, tag:client" />
                </label>
                <label>
                  <span>Destination</span>
                  <input bind:value={editDestination} placeholder="tag:web, db-host, 100.64.0.10" />
                </label>
                <label>
                  <span>Ports</span>
                  <select bind:value={editPortPreset}>
                    <option value="443">HTTPS 443</option>
                    <option value="80,443">HTTP/S 80,443</option>
                    <option value="22">SSH 22</option>
                    <option value="*">All ports</option>
                    <option value="custom">Custom</option>
                  </select>
                </label>
                {#if editPortPreset === "custom"}
                  <label>
                    <span>Custom ports</span>
                    <input bind:value={editCustomPorts} placeholder="8080,8443" />
                  </label>
                {/if}
                <div class="policy-actions">
                  <button class="secondary-btn" type="submit" disabled={editBusy}>Draft rule</button>
                  <button class="secondary-btn" type="button" onclick={validateDraft} disabled={editBusy || !draftHuJSON}>Validate</button>
                  <button class="primary-btn" type="button" onclick={saveDraft} disabled={editBusy || !draftValid}>Save</button>
                </div>
                {#if editStatus}
                  <p class="edit-status">{editStatus}</p>
                {/if}
              </form>
              {#if draftRuleText}
                <div class="draft-summary">
                  <p class="eyebrow">Rule to append</p>
                  <pre>{draftRuleText}</pre>
                </div>
              {/if}
              <details class="raw-policy" open={!draftHuJSON}>
                <summary>{draftHuJSON ? "Draft HuJSON" : "Current HuJSON"}</summary>
                <pre>{draftHuJSON || policy.hujson}</pre>
              </details>
            </div>
          </section>
        {/if}
        {#if cloudError}
          <div class="cloud-error" role="alert">{cloudError}</div>
        {/if}
      </section>

      <SidebarRight
        bind:open={rightOpen}
        bind:selectedDevice
        {apiVersion}
      />
    </div>
  </section>

  {#if phase2Open}
    <div class="dialog-backdrop" role="presentation">
      <div class="auth-dialog" role="dialog" aria-modal="true" aria-labelledby="phase2-title">
        <div class="auth-dialog-header">
          <div>
            <p class="eyebrow">Phase 2</p>
            <h2 id="phase2-title">Enable ACL Editing</h2>
          </div>
          <button type="button" title="Close" onclick={closePhase2Dialog}>×</button>
        </div>
        <form onsubmit={(event) => { event.preventDefault(); void enableACLEditing(); }}>
          <label>
            <span>Tailnet</span>
            <input bind:value={authTailnet} autocomplete="organization" placeholder="example.com or -" />
          </label>
          <label>
            <span>Tailscale API Key</span>
            <input bind:value={authAPIKey} autocomplete="off" type="password" placeholder="tskey-api-..." />
          </label>
          <p class="auth-note">
            The backend uses this key to fetch the policy file and keeps it in memory only.
          </p>
          {#if cloudError}
            <p class="form-error">{cloudError}</p>
          {/if}
          <div class="dialog-actions">
            <button class="secondary-btn" type="button" onclick={closePhase2Dialog} disabled={cloudBusy}>Cancel</button>
            <button class="primary-btn" type="submit" disabled={cloudBusy}>
              {cloudBusy ? "Connecting..." : "Fetch Policy"}
            </button>
          </div>
        </form>
      </div>
    </div>
  {/if}
</main>
