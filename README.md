# Shared Account AI Usage Monitor

**Current direction:** connect native coding-client sessions with people, projects, shared-account observations and coaching in one self-hosted workspace. The session reader is replaceable; see the [integration decision and migration gates](docs/agentsview-first.md). The features below describe the current prototype, not a completed migration.

An experimental, self-hosted team archive for native subscription coding-agent sessions. Search full available prompts, inspect recorded models and projects, discuss individual messages, and give **separate prompt and delivered-work ratings** with evidence.

The current local session reader integrates with [AgentsView](https://github.com/kenn-io/agentsview), an independent project by Kenn Software LLC. It is separately installed and maintained upstream; this project owns the team identity, policy, account observations and coaching layer. The integration boundary can evolve. See [third-party notices](THIRD_PARTY_NOTICES.md).

Public website and searchable docs: [usage.softinator.ai](https://usage.softinator.ai/), implemented independently in [`public-site/`](public-site/). The private team workspace runs at [usage.softinator.org](https://usage.softinator.org/).

## What works in the prototype

- People directory with email logins, roles, device OS/last connection and declared account assignments.
- Enrolled devices, explicit full/metadata collection and none/known-secret-pattern redaction.
- PostgreSQL archive, replay-safe revisions, private offline queue and chunked uploads.
- Email/password invitations, optional OIDC login, revocation, upload-only device tokens and read-only MCP tokens.
- Full-text search, source-aware conversations, human discussion, independent ratings and scoped external-agent drafts.
- Native Codex account/quota observations with reset times, device provenance and durable offline replay; declared assignments and manual observations remain separate.
- Source deletion, expiry and deletion-ledger replay on restore.
- Go binaries for companion/server/MCP, embedded browser UI and deployment examples.

No inference API gateway or paid inference API key is required. There is no conversion of token counts into a fictitious subscription bill. **Exact quota percentage per project/prompt is unknown without direct provider evidence.** Automatic Cockpit account switching/quotas, universal attachment fidelity, real Antigravity rollout and signed OS installers are not yet verified. See [delivery status](docs/status.md) and the [support matrix](compatibility/support-matrix.json).

## Local synthetic demo

Prerequisites: Go from go.mod and a disposable PostgreSQL database. Never reuse your production DSN for a demo/test.

```sh
export DATABASE_URL='postgresql://telemetry:test@127.0.0.1:5432/telemetry_demo?sslmode=disable'
go run ./cmd/team-server --demo --listen 127.0.0.1:8090
```

Open `http://127.0.0.1:8090`. Choose the synthetic owner/Alice/Bob personas. The example sessions are explicitly fictional. Demo auth refuses a public bind/origin; never reverse-proxy this mode publicly. For a custom port, set PUBLIC_ORIGIN to its exact loopback origin.

## Production and devices

See [deployment](docs/deployment.md), [device enrollment](docs/device-install.md) and [verification](docs/verification.md). Production needs your own HTTPS domain and a configured owner email. An OIDC application is optional. Full/no-redaction/team visibility is an explicit supported policy; the policy is shown to employees and version acknowledgement gates collection. Runtime prompts and credentials stay in private storage, never the public source repository.

## Ask your agent

Connect [read-only MCP](docs/coaching.md) to a native subscribed agent. Ask for evidence-grounded coaching in Hinglish or English. A model conclusion is a draft interpretation; prompt length, language fluency, tokens, LOC and commits are not employee productivity scores. Human review remains available for all employment/pay decisions.

[Architecture](docs/architecture.md) · [Agent harness](AGENTS.md) · [Contributing](CONTRIBUTING.md) · [Security](SECURITY.md) · [MIT license](LICENSE) · [Third-party notices](THIRD_PARTY_NOTICES.md)

### Understand usage shares

Open **People → a person → Today (IST)** for model/effort and recorded-project pies. Prompt counts and proposed edit lines are separate evidence. [Read the calculation and limits](docs/usage-estimates.md) before interpreting conditional subscription quota allocations: captured usage share is not the same as allowance consumed.
