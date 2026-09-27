import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./tests/public-site",
  workers: 1,
  reporter: "list",
  use: {
    baseURL: process.env.PUBLIC_SITE_ORIGIN || "http://127.0.0.1:18093",
    viewport: { width: 1440, height: 900 },
  },
});
