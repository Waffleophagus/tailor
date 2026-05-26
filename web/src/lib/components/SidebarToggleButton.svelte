<script lang="ts">
  let {
    open = true,
    position = "left",
    ontoggle,
  }: {
    open?: boolean;
    position?: "left" | "right";
    ontoggle?: () => void;
  } = $props();

  const arrow = $derived(open ? (position === "left" ? "←" : "→") : position === "left" ? "→" : "←");
</script>

<button
  class="sidebar-toggle"
  data-position={position}
  title={open ? "Collapse panel" : "Expand panel"}
  type="button"
  onclick={ontoggle}
>
  {arrow}
</button>

<style>
  .sidebar-toggle {
    position: absolute;
    top: 0.5rem;
    z-index: 10;
    display: grid;
    place-items: center;
    width: 1.7rem;
    height: 1.7rem;
    padding: 0;
    border: 1px solid #b9c5bf;
    border-radius: 6px;
    color: #586761;
    background: #ffffff;
    font-size: 0.75rem;
    font-weight: 700;
    line-height: 1;
    cursor: pointer;
    box-shadow: 0 2px 6px rgba(23, 33, 38, 0.06);
    transition:
      background-color 160ms ease-out,
      border-color 160ms ease-out,
      color 160ms ease-out,
      transform 160ms ease-out;
  }

  .sidebar-toggle[data-position="left"] {
    left: -0.85rem;
  }

  .sidebar-toggle[data-position="right"] {
    right: -0.85rem;
  }

  .sidebar-toggle:hover {
    border-color: #5d7f73;
    color: #18382d;
    background: #eef3f0;
    transform: scale(1.05);
  }

  @media (prefers-reduced-motion: reduce) {
    .sidebar-toggle {
      transition: none;
    }
  }
</style>
