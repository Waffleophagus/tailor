<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    position = "left",
    defaultWidth = 16 * 16,
    open = true,
    children,
    collapsed,
  }: {
    position?: "left" | "right";
    defaultWidth?: number;
    open?: boolean;
    children: Snippet;
    collapsed: Snippet;
  } = $props();

  const SIDEBAR_MIN = 12 * 16; // 12rem minimum
  const SIDEBAR_MAX = 30 * 16; // 30rem maximum
  const SIDEBAR_COLLAPSED = 2.75 * 16; // 2.75rem

  let sidebarWidth = $state(defaultWidth);
  let resizing = $state(false);

  function startResize(event: PointerEvent) {
    if (!open) return;
    resizing = true;
    const startX = event.clientX;
    const startWidth = sidebarWidth;

    function onMove(e: PointerEvent) {
      const delta = position === "left" ? e.clientX - startX : startX - e.clientX;
      sidebarWidth = Math.min(
        Math.max(startWidth + delta, SIDEBAR_MIN),
        SIDEBAR_MAX,
      );
    }

    function onUp() {
      resizing = false;
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    }

    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
  }
</script>

<div
  class="resizable-sidebar"
  data-position={position}
  data-open={open}
  data-resizing={resizing}
  style="width: {open ? sidebarWidth / 16 + 'rem' : SIDEBAR_COLLAPSED / 16 + 'rem'};"
>
  <div class="sidebar-content">
    {@render children()}
  </div>

  <!-- Resize handle -->
  <div
    class="resize-handle"
    role="separator"
    aria-label="Resize sidebar"
    data-resizing={resizing}
    onpointerdown={startResize}
  ></div>

  <!-- Collapsed icon bar -->
  <div class="icon-bar" aria-hidden={open}>
    <div class="icon-bar-content">
      {@render collapsed()}
    </div>
  </div>
</div>

<style>
  .resizable-sidebar {
    position: relative;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    background: #fbfcfb;
    transition: width 220ms cubic-bezier(0.4, 0, 0.2, 1);
    flex-shrink: 0;
  }

  .resizable-sidebar[data-resizing="true"] {
    transition: none;
  }

  .resizable-sidebar[data-open="false"] {
    width: 2.75rem !important;
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

  .resizable-sidebar[data-open="false"] .sidebar-content {
    opacity: 0;
    pointer-events: none;
    transition-delay: 0ms;
  }

  .resize-handle {
    position: absolute;
    top: 0;
    width: 4px;
    height: 100%;
    cursor: col-resize;
    z-index: 20;
    background: transparent;
    transition: background-color 160ms ease-out;
  }

  .resizable-sidebar[data-position="left"] .resize-handle {
    right: 0;
  }

  .resizable-sidebar[data-position="right"] .resize-handle {
    left: 0;
  }

  .resize-handle:hover,
  .resize-handle[data-resizing="true"] {
    background: #5d7f73;
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

  .resizable-sidebar[data-open="false"] .icon-bar {
    opacity: 1;
    pointer-events: auto;
  }

  .icon-bar-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 0.4rem;
  }
</style>
