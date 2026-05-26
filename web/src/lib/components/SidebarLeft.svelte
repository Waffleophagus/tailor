<script lang="ts">
  import type { Device } from "../api/schemas";
  import SearchInput from "./SearchInput.svelte";
  import SidebarToggleButton from "./SidebarToggleButton.svelte";

  let {
    open = $bindable(true),
    devices = [],
    visibleDevices = [],
    selectedDevice = $bindable<Device | undefined>(undefined),
    showLabels = $bindable(true),
    showOffline = $bindable(true),
    showSubnetRouters = $bindable(true),
    selectedTag = $bindable("all"),
    selectedOwner = $bindable("all"),
    selectedOS = $bindable("all"),
    colorBy = $bindable<"status" | "tag" | "owner" | "os">("status"),
    tagOptions = [],
    ownerOptions = [],
    osOptions = [],
    visibleOnlineCount = 0,
    chooseDevice,
  }: {
    open?: boolean;
    devices: Device[];
    visibleDevices: Device[];
    selectedDevice?: Device;
    showLabels?: boolean;
    showOffline?: boolean;
    showSubnetRouters?: boolean;
    selectedTag?: string;
    selectedOwner?: string;
    selectedOS?: string;
    colorBy?: "status" | "tag" | "owner" | "os";
    tagOptions: string[];
    ownerOptions: string[];
    osOptions: string[];
    visibleOnlineCount?: number;
    chooseDevice: (device: Device) => void;
  } = $props();

  let searchQuery = $state("");

  const filteredDevices = $derived(
    visibleDevices.filter((d) => {
      if (!searchQuery.trim()) return true;
      return d.name.toLowerCase().includes(searchQuery.toLowerCase().trim());
    }),
  );

  const visibleOfflineCount = $derived(visibleDevices.filter((d) => !d.online).length);
</script>

<div class="sidebar-left" data-open={open}>
  <div class="sidebar-content">
    <div class="sidebar-header">
      <h2>Devices</h2>
    </div>

    <div class="section">
      <h3 class="section-title">View</h3>
      <div class="control-grid">
        <label class="control-row">
          <input type="checkbox" bind:checked={showLabels} />
          <span>Show labels</span>
        </label>
        <label class="control-row">
          <input type="checkbox" bind:checked={showOffline} />
          <span>Show offline</span>
        </label>
        <label class="control-row">
          <input type="checkbox" bind:checked={showSubnetRouters} />
          <span>Subnet routers</span>
        </label>
      </div>
    </div>

    <div class="section">
      <h3 class="section-title">Filter</h3>
      <div class="filter-grid">
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
      </div>
    </div>

    <div class="section">
      <h3 class="section-title">Colorize</h3>
      <div class="segment-group">
        {#each (["status", "tag", "owner", "os"] as const) as mode}
          <button
            type="button"
            class="segment-btn"
            data-active={colorBy === mode}
            onclick={() => (colorBy = mode)}
          >
            {mode[0].toUpperCase() + mode.slice(1)}
          </button>
        {/each}
      </div>
    </div>

    <div class="section stretch">
      <h3 class="section-title">
        <span>List</span>
        <span class="status-pill" class:online={visibleOnlineCount > 0} class:offline={visibleOfflineCount > 0}>
          {visibleOnlineCount}/{visibleDevices.length}
        </span>
      </h3>
      {#if devices.length === 0}
        <p class="empty">No devices loaded.</p>
      {:else}
        <SearchInput
          bind:value={searchQuery}
          placeholder="Search devices..."
          count={filteredDevices.length}
          total={visibleDevices.length}
        />
        <ul class="device-list">
          {#each filteredDevices as device (device.id)}
            <li>
              <button
                class={["device-item", selectedDevice?.id === device.id && "active"]}
                type="button"
                onclick={() => chooseDevice(device)}
              >
                <span class={["dot", device.online && "online"]}></span>
                <span class="device-name">{device.name}</span>
              </button>
            </li>
          {/each}
        </ul>
      {/if}
    </div>
  </div>

  <!-- Collapsed icon bar -->
  <div class="icon-bar" aria-hidden={open}>
    <div class="icon-bar-content">
      <button
        class="icon-btn"
        title="Devices panel"
        type="button"
        onclick={() => (open = true)}
      >
        <svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path
            d="M10 2.5L2.5 7.5V17.5H8.75V12.5H11.25V17.5H17.5V7.5L10 2.5Z"
            fill="currentColor"
          />
        </svg>
      </button>
      <div class="icon-divider"></div>
      <div class="mini-counts">
        <span class="mini-count" title={`${visibleOnlineCount} online`}>
          <span class="mini-dot online"></span>
          {visibleOnlineCount}
        </span>
        <span class="mini-count" title={`${devices.length} total`}>
          {devices.length}
        </span>
      </div>
    </div>
  </div>
</div>

<style>
  .sidebar-left {
    position: relative;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: #fbfcfb;
    transition: width 220ms cubic-bezier(0.4, 0, 0.2, 1);
  }

  .sidebar-left[data-open="true"] {
    width: 16rem;
  }

  .sidebar-left[data-open="false"] {
    width: 2.75rem;
  }

  .sidebar-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-width: 0;
    padding: 1rem;
    overflow-y: auto;
    opacity: 1;
    transition: opacity 160ms ease-out 40ms;
  }

  .sidebar-left[data-open="false"] .sidebar-content {
    opacity: 0;
    pointer-events: none;
    transition-delay: 0ms;
  }

  .icon-bar {
    position: absolute;
    inset: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 0.5rem 0.25rem;
    opacity: 0;
    pointer-events: none;
    transition: opacity 140ms ease-out;
    background: #fbfcfb;
  }

  .sidebar-left[data-open="false"] .icon-bar {
    opacity: 1;
    pointer-events: auto;
  }

  .icon-bar-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.4rem;
  }

  .icon-btn {
    display: grid;
    place-items: center;
    width: 2rem;
    height: 2rem;
    padding: 0;
    border: 1px solid #d1dbd5;
    border-radius: 6px;
    color: #586761;
    background: transparent;
    cursor: pointer;
    transition:
      background-color 140ms ease-out,
      border-color 140ms ease-out,
      color 140ms ease-out;
  }

  .icon-btn:hover {
    border-color: #5d7f73;
    color: #18382d;
    background: #eef3f0;
  }

  .icon-btn svg {
    width: 1rem;
    height: 1rem;
  }

  .icon-divider {
    width: 1.2rem;
    height: 1px;
    background: #d9e1dd;
  }

  .mini-counts {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.3rem;
    margin-top: 0.2rem;
  }

  .mini-count {
    display: flex;
    align-items: center;
    gap: 0.2rem;
    font-size: 0.7rem;
    font-weight: 700;
    color: #586761;
    white-space: nowrap;
  }

  .mini-dot {
    width: 0.45rem;
    height: 0.45rem;
    border-radius: 999px;
    background: #9aa7a1;
  }

  .mini-dot.online {
    background: #41a86f;
  }

  .sidebar-header {
    flex-shrink: 0;
    margin-bottom: 0.75rem;
  }

  .sidebar-header h2 {
    margin: 0;
    font-size: 0.95rem;
    line-height: 1.2;
  }

  .section {
    flex-shrink: 0;
    padding-bottom: 0.85rem;
    margin-bottom: 0.85rem;
    border-bottom: 1px solid #e8eeeb;
  }

  .section:last-child,
  .section.stretch {
    flex: 1;
    flex-shrink: 1;
    margin-bottom: 0;
    border-bottom: 0;
    overflow-y: auto;
    min-height: 0;
  }

  .section-title {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin: 0 0 0.5rem 0;
    padding: 0;
    color: #3a4a44;
    font-size: 0.72rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .status-pill {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    padding: 0.15rem 0.4rem;
    border-radius: 999px;
    font-size: 0.68rem;
    font-weight: 700;
    background: #edf4f0;
    color: #5d7f73;
    transition: background-color 160ms ease-out, color 160ms ease-out;
  }

  .status-pill.online {
    background: #e8f5ee;
    color: #2a9d5e;
  }

  .status-pill.offline {
    background: #f0f2f1;
    color: #8a9590;
  }

  .control-grid {
    display: flex;
    flex-direction: column;
    gap: 0.35rem;
  }

  .control-row {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    color: #2f3c37;
    font-size: 0.85rem;
    font-weight: 700;
    cursor: pointer;
  }

  .control-row input {
    width: 1rem;
    height: 1rem;
    margin: 0;
  }

  .filter-grid {
    display: flex;
    flex-direction: column;
    gap: 0.45rem;
  }

  .filter-grid label {
    display: grid;
    grid-template-columns: 3.5rem minmax(0, 1fr);
    align-items: center;
    gap: 0.4rem;
    color: #2f3c37;
    font-size: 0.85rem;
    font-weight: 700;
  }

  .filter-grid select {
    width: 100%;
    min-width: 0;
    padding: 0.35rem 0.45rem;
    border: 1px solid #b9c5bf;
    border-radius: 6px;
    color: #172126;
    background: #ffffff;
    font-size: 0.85rem;
    outline: none;
    transition: border-color 140ms ease-out, box-shadow 140ms ease-out;
  }

  .filter-grid select:focus {
    border-color: #5d7f73;
    box-shadow: 0 0 0 3px rgba(93, 127, 115, 0.12);
  }

  .segment-group {
    display: flex;
    gap: 0;
  }

  .segment-btn {
    flex: 1;
    padding: 0.35rem 0.2rem;
    border: 1px solid #c7d3ce;
    border-radius: 0;
    color: #586761;
    background: #f3f6f4;
    font-size: 0.78rem;
    font-weight: 700;
    cursor: pointer;
    white-space: nowrap;
    transition:
      background-color 140ms ease-out,
      border-color 140ms ease-out,
      color 140ms ease-out;
  }

  .segment-btn:first-child {
    border-radius: 6px 0 0 6px;
  }

  .segment-btn:last-child {
    border-radius: 0 6px 6px 0;
  }

  .segment-btn[data-active="true"] {
    border-color: #5d7f73;
    color: #18382d;
    background: #eef3f0;
  }

  .segment-btn:hover:not([data-active="true"]) {
    background: #e8eeeb;
    color: #2f3c37;
  }

  .device-list {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    padding: 0;
    margin: 0.6rem 0 0 0;
    list-style: none;
    overflow-y: auto;
  }

  .device-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    width: 100%;
    padding: 0.45rem 0.5rem;
    border: 1px solid transparent;
    border-radius: 6px;
    color: #172126;
    background: transparent;
    font-size: 0.85rem;
    text-align: left;
    cursor: pointer;
    transition:
      background-color 140ms ease-out,
      border-color 140ms ease-out;
  }

  .device-item:hover,
  .device-item.active {
    border-color: #c7d3ce;
    background: #eef3f0;
  }

  .device-name {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .dot {
    flex-shrink: 0;
    width: 0.6rem;
    height: 0.6rem;
    border-radius: 999px;
    background: #9aa7a1;
  }

  .dot.online {
    background: #41a86f;
  }

  .empty {
    margin-top: 0.5rem;
    color: #98a8a0;
    font-size: 0.85rem;
  }
</style>
