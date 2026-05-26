<script lang="ts">
  import { onMount } from "svelte";

  import { fetchHealth } from "./lib/api/health";

  let apiStatus = $state("checking");
  let apiVersion = $state("");

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
  });
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
        <p>LocalAPI discovery will populate this panel.</p>
      </aside>

      <section class="graph" aria-label="Topology graph canvas placeholder">
        <div class="node node-a">client</div>
        <div class="node node-b">server</div>
        <div class="edge"></div>
      </section>

      <aside>
        <h2>Policy lens</h2>
        <p>Selected devices will show tags, owner, and matching policy subjects.</p>
        <p class="meta">API version: {apiVersion || "unknown"}</p>
      </aside>
    </div>
  </section>
</main>
