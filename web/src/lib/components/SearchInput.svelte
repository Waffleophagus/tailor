<script lang="ts">
  import type { Device } from "../api/schemas";

  let {
    value = $bindable(""),
    placeholder = "Search...",
    count = 0,
    total = 0,
  }: {
    value?: string;
    placeholder?: string;
    count?: number;
    total?: number;
  } = $props();
</script>

<div class="search-wrap">
  <svg class="search-icon" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
    <path
      d="M17.5 17.5L13.875 13.875M15.8333 9.16667C15.8333 12.8486 12.8486 15.8333 9.16667 15.8333C5.48477 15.8333 2.5 12.8486 2.5 9.16667C2.5 5.48477 5.48477 2.5 9.16667 2.5C12.8486 2.5 15.8333 5.48477 15.8333 9.16667Z"
      stroke="currentColor"
      stroke-width="1.6"
      stroke-linecap="round"
      stroke-linejoin="round"
    />
  </svg>
  <input
    type="text"
    {placeholder}
    bind:value
    aria-label="Search devices"
  />
  {#if value}
    <button class="clear" type="button" title="Clear search" onclick={() => (value = "")}>
      <svg viewBox="0 0 14 14" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path d="M1 1L13 13M13 1L1 13" stroke="currentColor" stroke-width="1.4" stroke-linecap="round" />
      </svg>
    </button>
  {/if}
</div>
<p class="result-count">{count} of {total} devices visible</p>

<style>
  .search-wrap {
    position: relative;
    display: flex;
    align-items: center;
  }

  .search-icon {
    position: absolute;
    left: 0.55rem;
    width: 0.9rem;
    height: 0.9rem;
    pointer-events: none;
    color: #8ba095;
  }

  input {
    width: 100%;
    height: 2rem;
    padding: 0 1.75rem 0 1.7rem;
    border: 1px solid #b9c5bf;
    border-radius: 6px;
    color: #172126;
    background: #ffffff;
    font-size: 0.85rem;
    outline: none;
    transition:
      border-color 140ms ease-out,
      box-shadow 140ms ease-out;
  }

  input:focus {
    border-color: #5d7f73;
    box-shadow: 0 0 0 3px rgba(93, 127, 115, 0.12);
  }

  input::placeholder {
    color: #98a8a0;
  }

  .clear {
    position: absolute;
    right: 0.35rem;
    display: grid;
    place-items: center;
    width: 1.4rem;
    height: 1.4rem;
    padding: 0;
    border: none;
    border-radius: 4px;
    color: #98a8a0;
    background: transparent;
    cursor: pointer;
    transition: color 120ms ease-out;
  }

  .clear:hover {
    color: #586761;
  }

  .clear svg {
    width: 0.7rem;
    height: 0.7rem;
  }

  .result-count {
    margin-top: 0.4rem;
    margin-bottom: 0;
    color: #596963;
    font-size: 0.78rem;
    font-weight: 700;
    line-height: 1.3;
  }
</style>
