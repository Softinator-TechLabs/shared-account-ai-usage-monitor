# Verification

Use an explicit disposable PostgreSQL database, never a production DSN. Tests create random schemas and remove only their own schemas. Go is pinned by `toolchain` in go.mod. Python tests use the standard library.

```
python3 -m unittest discover -s tests -p 'test_*.py' -v
TEST_DATABASE_URL=postgresql://telemetry:test@127.0.0.1:5432/telemetry_test go test -race ./... -count=1
go vet ./...
node --check internal/web/assets/app.js
npm ci
npx playwright install chromium
# Start the loopback demo according to README first:
npm run test:ui
```

Synthetic browser scenarios cover login, search, full conversation, anchored prompt rating and mobile layout. Store/HTTP tests cover replay/conflicts, Unicode fidelity, visibility, revocation, policy acknowledgements, CSRF, verified OIDC callback/nonces, chunks, deletion/restore, manual quota bounds and scoped agent drafts. The browser runner is development-only.

For the actual official AgentsView → companion → central server → MCP path, start a separate AgentsView data directory containing only the supplied synthetic fixture. Pass its explicit origin and token path to `scripts/smoke.py`; the smoke refuses an upstream listing that includes non-synthetic session IDs. Build the binaries first. A provider login is not required for parsing this synthetic fixture.

```
go build -o .local/team-agent ./cmd/team-agent
go build -o .local/team-mcp ./cmd/team-mcp
python3 scripts/smoke.py --base-url http://127.0.0.1:8090 \
 --agent-binary "$PWD/.local/team-agent" --mcp-binary "$PWD/.local/team-mcp" \
 --upstream http://127.0.0.1:18086 --upstream-token-file /private/synthetic-token
```

A successful source/unit/browser/cross-build check is not a signed Windows/macOS installer, production OIDC deployment, real employee acknowledgement, Antigravity full-fidelity proof or live Dokploy receipt. Record those separately in compatibility/support-matrix.json and docs/status.md.
