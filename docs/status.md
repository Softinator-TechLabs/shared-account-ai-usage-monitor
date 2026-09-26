# Delivery status · 2026-09-26

Status: experimental implementation under active verification, not a production rollout.

Implemented: Session API adapter, policy transformation, PostgreSQL archive/revisions, hashed single-use enrollment, email/password invitation setup, employee/device directory and connection heartbeat, optional OIDC state/nonce/PKCE verification, full-text search/export, immutable message discussions, independent ratings, externally submitted agent drafts, account declarations/manual quota history, read-only MCP, chunks, expiry/deletion/restore ledger, embedded responsive UI and deployment documentation.

Verified so far: synthetic official AgentsView v0.44.0 macOS arm64 preserves a 29,000-character Codex prompt; PostgreSQL integration/HTTP tests; synthetic browser owner login/search/prompt rating/mobile navigation. Exact final command receipts will be appended after final verification.

Not yet certified: real employees/acknowledgements; Windows/Ubuntu native startup/sleep/wake; Antigravity full history/attachments; Cockpit profile/account-switch attribution; automatic provider quotas/resets; actual native analysis executor/model provenance; production OIDC tenant; container/Dokploy/DNS/TLS; public clean-tree release. No paid hours, exact project quota percentage or universal full telemetry claim is made.

Known engineering limits: 15-minute metadata reconciliation with per-source transcript-revision checkpoints; queue and response bounds fail visibly; collection uses acknowledged cached policy offline; stale lock recovery is manual; source copies retain unknown historical actor and lack a per-device delivery ledger; search currently uses PostgreSQL ILIKE rather than a large-corpus index; equal-timestamp pagination needs a composite cursor before high-volume deployments; all immutable captures remain searchable and upload order is explicitly not source chronology; provider tokens/timing are raw evidence, not normalized billing.

Local native→companion→archive→MCP smoke passed on 2026-09-26 using an isolated official AgentsView instance and a 29,000-character synthetic prompt. Six browser workflows passed (including session permalink reload and seven-day read-only agent login/revocation): review flow on desktop/mobile, and owner creates email user → user sets password → member login. Live deployment remains a separate gate.

The latest desktop sidebar covers the complete document and remains viewport-height while scrolling; profile/sign-out visibility was checked at the page bottom. The intermediate-width rail uses the same 200px surface and offset.

The bounded whole-branch review identified public-export, deletion queue, initial queue drain, UTF-8 fidelity, offline revision discovery, revoked capture and prior-discussion discoverability issues. Regression fixtures reproduced the failures. Repairs use fixed-commit export, terminal deletion suppression, streamed collection with pre-drain, byte validation before decoding, visible immutable captures, and structured offline-versus-denied responses. A moving native transcript is checked against its released source revision before acknowledgement. Final browser and race receipts are separate from live/device evidence.
