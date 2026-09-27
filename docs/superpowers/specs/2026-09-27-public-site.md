# Public website and docs · 2026-09-27

The owner wants a separate, expressive open-source website on `usage.softinator.ai`, with documentation and GitHub downloads as discoverable as the Feedbacks website. It must run as an independent Dokploy project from the same public product repository. The existing private workspace stays at `usage.softinator.org`.

The public experience has two modes: a succinct landing page for visitors deciding whether to try the product, and searchable technical documentation for hosting, installing and connecting it. The landing page starts with a self-hosting path, gives direct Mac releases and GitHub source, then shows an explicitly illustrative account/session/project view. The docs include Docker/Dokploy setup, first-owner bootstrap, platform-specific availability, the actual connection-file steps, privacy policy controls, analytics meaning and limitations. Neither surface fetches private workspace data or routes new users into Softinator's private workspace.

The public build is a standalone static subtree. Vite assembles the landing page and VitePress assembles Markdown docs under `/docs/`. Nginx serves both from an unprivileged container. Dokploy uses a distinct project, compose service and HTTPS domain, leaving the private app/database untouched. Build and smoke checks cover navigation, release links, responsive layout, documentation, security headers and health.

The visual language uses ink-dark foundations, electric citron, lavender and warm off-white. A custom signal-board illustration communicates the relationship between accounts, people, sessions and projects without claiming numerical accuracy. Motion is restrained, keyboard focus is visible, reduced-motion works, and the hierarchy remains clear on narrow screens.

AgentsView is one current independently maintained local reader/source. Keep required licensing and a precise integration note, but do not imply the product is permanently built on or owned by it. Do not promote upstream credit in either footer. Factual product claims must follow `docs/status.md` and `docs/usage-estimates.md`; no exact personal quota, billing or productivity claim.
