import { defineConfig } from "vite";

export default defineConfig({
  root: "public-site/site",
  build: { outDir: "../dist/website", emptyOutDir: true },
  server: { host: "127.0.0.1", port: 4175, strictPort: true },
});
