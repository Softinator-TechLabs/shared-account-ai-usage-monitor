# Public website and docs

This directory builds the independent public site at `usage.softinator.ai`. It contains no team account credentials, prompts or telemetry. The private workspace at `usage.softinator.org` is deployed from `deploy/compose.yml` and does not share this site's runtime or database.

## Work locally

From the repository root, run `npm ci`, then `npm run dev:public-site` for the landing page or `npm run dev:public-docs` for the documentation. `npm run build:public-site` writes the combined static output to `public-site/dist/website/` and exports every guide as Markdown along with `llms.txt`. `npm run test:public-site` checks the built site through an HTTP server; CI runs it against the production Nginx container.

## Publish

Build `public-site/Dockerfile`, or deploy `public-site/compose.yml` from the repository root as the Git source. Route the public domain to the `website` service on port `8080`. The site needs no secrets or database. The health route is `/healthz`; the documentation lives under `/docs/`.

The live Dokploy service is a separate **Shared Account AI Usage Monitor Website** project. Deploy it after changes land on `main`, then check `/`, `/docs/`, `/docs/guide/self-host`, `/docs/guide/getting-started`, `/docs/llms.txt` and the Mac release links. The public site must never send visitors to Softinator's private team workspace as a setup shortcut. The release links in `site/index.html` and the docs must be updated together when a new installer is published. Claims about platform support and quota attribution must match [`docs/status.md`](../docs/status.md) and [`docs/usage-estimates.md`](../docs/usage-estimates.md).

The current local reader's third-party attribution is maintained in [`THIRD_PARTY_NOTICES.md`](../THIRD_PARTY_NOTICES.md) and the [integration guide](docs/guide/integrations.md). This site describes the product independently of any one reader.
