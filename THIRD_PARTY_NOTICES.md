# Third-party notices

Built with [AgentsView](https://github.com/kenn-io/agentsview) integration. AgentsView is an independent open-source project by Kenn Software LLC. This project adds team identity, subscription-account observations, transparent access policies and evidence-based coaching. It is not an official AgentsView product.

AgentsView is installed separately and accessed through its Session API. No upstream binary or parser source is distributed here. Its inspected release is v0.44.0; see `compatibility/agentsview.json`. Its MIT copyright notice remains applicable to any separately redistributed upstream artifacts.

Geist Sans is distributed under the SIL Open Font License 1.1. Exact notice: `internal/web/assets/GEIST-LICENSE.txt`. Font source: https://github.com/vercel/geist-font (release branch, downloaded 2026-09-26).

Go dependencies are pinned in go.mod/go.sum. pgx/pgpassfile/pgservicefile/puddle use MIT; go-oidc uses Apache-2.0; go-jose uses Apache-2.0; golang.org/x packages use BSD-3-Clause. A release must retain the corresponding notices with compiled distributions. `scripts/dependency_notices.py` collects their upstream notice files from the downloaded module cache; generated files are release artifacts, not manually edited copies.

Playwright is a development-only Apache-2.0 dependency. Container base images retain their respective distribution licenses and package notices.

Geist font source commit: `10dc7658f13c38a474cde201bb09a4617267545b`. See [asset provenance](docs/ui-assets.md).
