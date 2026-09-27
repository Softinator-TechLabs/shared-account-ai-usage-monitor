import { defineConfig } from "vitepress";

export default defineConfig({
  title: "Shared AI Usage Docs",
  description:
    "Install, connect and interpret Shared Account AI Usage Monitor.",
  lang: "en-US",
  base: "/docs/",
  cleanUrls: true,
  outDir: "../dist/website/docs",
  sitemap: { hostname: "https://usage.softinator.ai" },
  head: [
    ["meta", { name: "theme-color", content: "#171b20" }],
    ["link", { rel: "icon", href: "/mark.svg" }],
  ],
  themeConfig: {
    logo: "/mark.svg",
    siteTitle: "Shared AI Usage",
    nav: [
      { text: "Website", link: "https://usage.softinator.ai/" },
      { text: "Get started", link: "/guide/getting-started" },
      { text: "How numbers work", link: "/guide/usage-and-limits" },
      { text: "Workspace ↗", link: "https://usage.softinator.org/" },
    ],
    sidebar: [
      {
        text: "Start here",
        items: [
          { text: "Overview", link: "/" },
          { text: "Getting started", link: "/guide/getting-started" },
          { text: "Connect a device", link: "/guide/connect-device" },
        ],
      },
      {
        text: "Understand the data",
        items: [
          { text: "Usage and limits", link: "/guide/usage-and-limits" },
          { text: "Privacy and policy", link: "/guide/privacy-and-policy" },
          { text: "Current support", link: "/guide/current-support" },
        ],
      },
      {
        text: "Run it yourself",
        items: [
          { text: "Self-host", link: "/guide/self-host" },
          { text: "Integrations", link: "/guide/integrations" },
        ],
      },
    ],
    search: { provider: "local" },
    editLink: {
      pattern:
        "https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/edit/main/public-site/docs/:path",
      text: "Improve this page",
    },
    socialLinks: [
      {
        icon: "github",
        link: "https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor",
      },
    ],
    footer: {
      message: "Open-source documentation for Shared Account AI Usage Monitor.",
      copyright: "© Softinator TechLabs",
    },
  },
});
