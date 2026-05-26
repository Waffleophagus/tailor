import { defineConfig } from "@hey-api/openapi-ts";

export default defineConfig({
  input: "../docs/openapi.yaml",
  output: "src/lib/api/client",
  plugins: ["@hey-api/client-fetch"],
});
