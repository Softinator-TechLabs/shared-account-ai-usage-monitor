# Decisions

- **2026-09-26 · central upstream reuse supersedes custom reader expansion.** Reuse independently versioned AgentsView PostgreSQL sync/UI after access, fidelity, device and rollback gates. Keep our identity/account/quota/coaching layer. Existing archive remains supported during migration; see [accepted design](agentsview-first.md). This supersedes the original Session-API-only target below, not its current runtime or evidence safeguards.

- **2026-09-26 · composition over parser fork.** Use official AgentsView Session API. One adapter owns API drift; fixture/canary gates replace blind updates. No maintenance-free guarantee.
- **Go/PostgreSQL/embedded UI.** One module, three deployable processes, no frontend production runtime. Python stdlib research/compatibility scripts and Node browser tests are development tooling.
- **Revision snapshot as atomic record.** Preserve raw API JSON plus normalized fields. Message anchors are revision+ordinal; no invented native UUID. API index rebuilds can create new revisions. Per-device duplicate delivery provenance remains a follow-up.
- **Explicit owner policy.** Full/no-redaction/team is allowed. Metadata and known-secret-pattern modes share device/server transformation. Policy changes cannot silently relabel old queues.
- **Subscription accounting honesty.** Manual account observations first. Token data stays source-described; no guessed per-project percentage, API-equivalent bill or productivity leaderboard.
- **Scoped external-agent draft.** Native analysis stays with the subscribed local agent. Central result capability expires, attaches to one revision/message, and cannot set human ratings. A provider-native executor is a later compatibility gate, not an unrestricted server subprocess.
- **Public code/private operations.** Release only a clean allowlisted product tree with new history. Never change the private management repository's visibility. No real roster, archive, credentials or deployment configuration in public artifacts.
- **Bootstrap versus release evidence.** Development PostgreSQL 15.13 remains a local receipt. Updated Go toolchain/dependencies require rerun tests. Container tags are examples until actual container and deployment verification.

- **Ambiguous source chronology.** Receipt order cannot select a reliable native “latest” across offline devices. Expose/search all immutable captures and exact-source capture history, clearly labelled by receipt time; preserve old discussion anchors. This costs additional capture rows and avoids hiding evidence.
- **Release export is a fixed Git tree.** Export only committed product blobs from an exact commit, and reject tracked runtime/credential paths. Ignored, untracked and modified working files do not enter the export.
- **Named debugging access.** Seven-day owner-issued links grant read-only browser/API access, expire without renewal and can be revoked. They cannot post human ratings or change users/policy.

- **2026-09-26 · released PG push fails the device-writer gate.** Synthetic official v0.44.0 push works and repeated pushes keep one session. Its ordinary shared-schema DML role can read/update every session; the schema has no row-level isolation. Do not distribute that role to employee devices. Continue independent account/quota delivery while evaluating a scoped transport. Released artifact-folder sync is experimentally available but documented for trusted folders; it is not silently promoted into a production HTTP upload/deletion contract. Central cutover remains gated; existing reviewed captures are preserved.
