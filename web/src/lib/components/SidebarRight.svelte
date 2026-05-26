<script lang="ts">
  import type { Device } from "../api/schemas";

  let {
    open = $bindable(true),
    selectedDevice = $bindable<Device | undefined>(undefined),
    apiVersion = "",
  }: {
    open?: boolean;
    selectedDevice?: Device;
    apiVersion?: string;
  } = $props();

  const deviceInitials = $derived(
    selectedDevice?.name
      ? selectedDevice.name
          .split(".")[0]
          .slice(0, 2)
          .toUpperCase()
      : "?",
  );
</script>

<div class="sidebar-right" data-open={open}>
  <div class="sidebar-content">
    <div class="sidebar-header">
      <h2>Policy Lens</h2>
    </div>

    {#if selectedDevice}
      <div class="device-header">
        <span class="device-avatar" class:online={selectedDevice.online} data-subnet-router={selectedDevice.subnetRouter}>
          {deviceInitials}
        </span>
        <div class="device-meta">
          <p class="device-name" title={selectedDevice.name}>{selectedDevice.name}</p>
          <div class="device-status">
            <span class="dot" class:online={selectedDevice.online}></span>
            {selectedDevice.online ? "online" : "offline"}
            {#if selectedDevice.tags.length > 0}
              <span class="tag-pill">{selectedDevice.tags[0]}</span>
              {#if selectedDevice.tags.length > 1}
                <span class="tag-more" title={selectedDevice.tags.slice(1).join(", ")}>
                  +{selectedDevice.tags.length - 1}
                </span>
              {/if}
            {/if}
          </div>
        </div>
      </div>
    {/if}

    <div class="auth-banner">ACL editing requires Phase 2 authentication.</div>

    {#if selectedDevice}
      <div class="details">
        <div class="detail-section">
          <h3 class="section-title">Identity</h3>
          <div class="detail-row">
            <span class="detail-label">Name</span>
            <span class="detail-value">{selectedDevice.name}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Owner</span>
            <span class="detail-value">{selectedDevice.owner || "unknown"}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">OS</span>
            <span class="detail-value">{selectedDevice.os || "unknown"}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Status</span>
            <span class="detail-value">
              <span class="dot" class:online={selectedDevice.online}></span>
              {selectedDevice.online ? "online" : "offline"}
            </span>
          </div>
        </div>

        <div class="detail-section">
          <h3 class="section-title">Network</h3>
          <div class="detail-row">
            <span class="detail-label">IP</span>
            <span class="detail-value">{selectedDevice.ip || "unknown"}</span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Tailscale IPs</span>
            <span class="detail-value">
              {selectedDevice.tailscaleIps.length
                ? selectedDevice.tailscaleIps.join(", ")
                : "unknown"}
            </span>
          </div>
          <div class="detail-row">
            <span class="detail-label">Subnet routes</span>
            <span class="detail-value">
              {selectedDevice.routedSubnets.length
                ? selectedDevice.routedSubnets.join(", ")
                : "none"}
            </span>
          </div>
        </div>

        <div class="detail-section">
          <h3 class="section-title">Tags</h3>
          {#if selectedDevice.tags.length > 0}
            <div class="tag-list">
              {#each selectedDevice.tags as tag}
                <span class="tag-pill large">{tag}</span>
              {/each}
            </div>
          {:else}
            <p class="empty-value">none</p>
          {/if}
        </div>
      </div>
    {:else}
      <div class="empty-state">
        <p class="empty-hint">Select a node in the graph to inspect its details.</p>
      </div>
    {/if}

    <p class="meta">API version: {apiVersion || "unknown"}</p>
  </div>

  <!-- Collapsed icon bar -->
  <div class="icon-bar" aria-hidden={open}>
    <div class="icon-bar-content">
      <button
        class="icon-btn"
        title="Policy Lens panel"
        type="button"
        onclick={() => (open = true)}
      >
        <svg viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path
            d="M10 1L1 5V10C1 15.25 4.75 19.35 10 20C15.25 19.35 19 15.25 19 10V5L10 1Z"
            stroke="currentColor"
            stroke-width="1.6"
            stroke-linecap="round"
            stroke-linejoin="round"
            fill="none"
          />
        </svg>
      </button>
      <div class="icon-divider"></div>
      {#if selectedDevice}
        <button
          class="mini-device"
          title={selectedDevice.name}
          type="button"
          onclick={() => (open = true)}
        >
          <span class="mini-avatar" class:online={selectedDevice.online} data-subnet-router={selectedDevice.subnetRouter}>{deviceInitials}</span>
        </button>
      {:else}
        <span class="mini-hint" title="No device selected">—</span>
      {/if}
    </div>
  </div>
</div>

<style>
  .sidebar-right {
    position: relative;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: #fbfcfb;
    transition: width 220ms cubic-bezier(0.4, 0, 0.2, 1);
  }

  .sidebar-right[data-open="true"] {
    width: 18rem;
  }

  .sidebar-right[data-open="false"] {
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

  .sidebar-right[data-open="false"] .sidebar-content {
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

  .sidebar-right[data-open="false"] .icon-bar {
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

  .mini-device {
    display: grid;
    place-items: center;
    width: 2rem;
    height: 2rem;
    padding: 0;
    border: none;
    border-radius: 8px;
    background: transparent;
    cursor: pointer;
  }

  .mini-avatar {
    display: grid;
    place-items: center;
    width: 1.8rem;
    height: 1.8rem;
    border-radius: 999px;
    color: #ffffff;
    background: #9aa7a1;
    font-size: 0.65rem;
    font-weight: 700;
    transition: background-color 160ms ease-out;
  }

  .mini-avatar.online {
    background: #41a86f;
  }

  .mini-avatar[data-subnet-router="true"] {
    border-radius: 8px;
  }

  .mini-hint {
    color: #98a8a0;
    font-size: 0.75rem;
    font-weight: 700;
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

  .device-header {
    display: flex;
    align-items: center;
    gap: 0.65rem;
    padding-bottom: 0.85rem;
    margin-bottom: 0.85rem;
    border-bottom: 1px solid #e8eeeb;
  }

  .device-avatar {
    flex-shrink: 0;
    display: grid;
    place-items: center;
    width: 2.25rem;
    height: 2.25rem;
    border-radius: 999px;
    color: #ffffff;
    background: #9aa7a1;
    font-size: 0.8rem;
    font-weight: 700;
    transition: background-color 160ms ease-out;
  }

  .device-avatar.online {
    background: #41a86f;
  }

  .device-avatar[data-subnet-router="true"] {
    border-radius: 8px;
  }

  .device-meta {
    min-width: 0;
  }

  .device-name {
    margin: 0;
    overflow: hidden;
    color: #172126;
    font-size: 0.9rem;
    font-weight: 700;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .device-status {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    margin-top: 0.15rem;
    color: #596963;
    font-size: 0.78rem;
    font-weight: 700;
  }

  .auth-banner {
    padding: 0.65rem 0.7rem;
    margin-bottom: 1rem;
    border: 1px solid #d8c397;
    border-radius: 8px;
    color: #59411c;
    background: #fff8e6;
    font-size: 0.85rem;
    font-weight: 700;
    line-height: 1.35;
  }

  .details {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    flex: 1;
    overflow-y: auto;
  }

  .detail-section {
    padding-bottom: 0.85rem;
    border-bottom: 1px solid #e8eeeb;
  }

  .detail-section:last-child {
    border-bottom: 0;
    padding-bottom: 0;
  }

  .section-title {
    margin: 0 0 0.5rem 0;
    padding: 0;
    color: #3a4a44;
    font-size: 0.72rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .detail-row {
    display: grid;
    grid-template-columns: 6rem minmax(0, 1fr);
    gap: 0.5rem;
    align-items: baseline;
    padding: 0.25rem 0;
  }

  .detail-label {
    color: #596963;
    font-size: 0.78rem;
    font-weight: 700;
  }

  .detail-value {
    display: flex;
    align-items: center;
    gap: 0.3rem;
    overflow-wrap: anywhere;
    color: #172126;
    font-size: 0.85rem;
  }

  .empty-value {
    margin: 0;
    color: #98a8a0;
    font-size: 0.85rem;
  }

  .dot {
    display: inline-block;
    width: 0.6rem;
    height: 0.6rem;
    border-radius: 999px;
    background: #9aa7a1;
  }

  .dot.online {
    background: #41a86f;
  }

  .tag-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.35rem;
  }

  .tag-pill {
    display: inline-flex;
    align-items: center;
    padding: 0.15rem 0.45rem;
    border-radius: 999px;
    color: #2f3c37;
    background: #e8eeeb;
    font-size: 0.75rem;
    font-weight: 700;
  }

  .tag-pill.large {
    padding: 0.25rem 0.55rem;
    font-size: 0.8rem;
  }

  .tag-more {
    color: #98a8a0;
    font-size: 0.75rem;
    font-weight: 700;
  }

  .empty-state {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 2rem 0;
  }

  .empty-hint {
    margin: 0;
    color: #98a8a0;
    font-size: 0.85rem;
    text-align: center;
    line-height: 1.5;
  }

  .meta {
    margin-top: 1.5rem;
    margin-bottom: 0;
    padding-top: 0.75rem;
    border-top: 1px solid #e8eeeb;
    color: #315044;
    font-size: 0.8rem;
    font-weight: 700;
  }
</style>
