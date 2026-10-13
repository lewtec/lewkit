import { defineConfig } from "astro/config";
import cloudflare from "@astrojs/cloudflare";

// SSR: frontmatter runs per request in the orvalho workers isolate.
// Same CF adapter generation path as pesquisarr (Astro 6 + @astrojs/cloudflare 13).
export default defineConfig({
  output: "server",
  adapter: cloudflare({
    platformProxy: { enabled: false },
  }),
  outDir: "dist",
});
