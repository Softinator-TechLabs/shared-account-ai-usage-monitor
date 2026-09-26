# Architecture and evidence boundaries

The product is one Go module. Three deployable binaries: `team-server`, `team-agent`, and `team-mcp`. PostgreSQL is the only central runtime dependency. Browser dependencies are test-only. Install official AgentsView separately; its native parsers are not forked or copied.

```
Native sessions → local AgentsView Session API → enrolled companion + private queue
                                                  ↓ HTTPS
                                        central Go server → PostgreSQL
                                                  ↑
                             browser / permissioned read-only MCP
```

The `agentsview` package alone owns upstream API shapes. Unknown JSON fields survive in `raw`. A session snapshot is an immutable revision; message ordinals are anchored to that revision, not claimed as universal native UUIDs. Rebuilding an upstream index may change database-local IDs and produce a new revision. Copies may share a source reference; the first observed owner is never proof of a historical actor. This prototype does not yet retain a separate per-device delivery ledger for every duplicate.

`policy.Apply` runs before local queueing and again before central storage. Full/none preserves all content supplied by the API. Metadata removes message/raw content. Secret mode handles documented token-shaped patterns only, not arbitrary PII or every possible secret. Policy changes stop uploads until explicit acknowledgement; old queued versions must be delivered under their still-current policy or explicitly discarded and recollected. No automatic relabeling of historical content.

Human authentication uses verified OIDC issuer/audience, state, nonce, PKCE and opaque database-backed cookies. Device invitations are single-use, expire in 24 hours and bind member/device/policy. Device tokens expire after three months; portal tokens after eight hours. Revoking a member revokes all their tokens. Read tokens reject mutations. Device tokens cannot read the team archive or administer it. Only loopback synthetic demo mode bypasses OIDC.

Ingest is idempotent by workspace/source/revision and digest. Large records use 1 MiB persisted chunks and an atomic archive commit; incomplete uploads never appear in search. Supported central record size is at most 512 MiB; the current upstream API reader limits each response page to 128 MiB and fails visibly rather than truncating it. Queue capacity defaults to 1 GiB and never evicts unacknowledged records. A crash may require removal of a stale collector lock after verifying no process is alive.

Search, direct reads, exports and reviews share server-side permissions. Text is rendered with `textContent`, never HTML. Read receipts exclude transcript contents. Source deletion cascades to comments and analysis runs, clears staging, and leaves a tombstone. Automatic retention uses archive receipt time. Backup restore must replay a newer separately retained deletion ledger before the server becomes accessible.

Account assignments are declarations, not usage proof. Quota observations are timestamped manual observations in this release. Provider-specific automatic quota scraping, account-switch detection and exact prompt/project percentage allocation are not implemented. Unknown fields stay unknown; there is no fake API-dollar billing.

Agent analysis runs are one-session, one-message, one-hour, one-result capabilities; results are labeled external agent drafts with unverified model identity. They cannot change ratings or publish employee scores. A human adds independent prompt/work ratings, with work evidence mandatory. The read-only MCP bridge cannot submit comments.

## Capture ordering and collection lifecycle

The archive exposes immutable captures, including older captures with discussions. Server receipt time is displayed as receipt time; it is not native chronology. Search covers every permitted capture. The session view offers “See captures & discussions” for exact-source filtering. Reviews always retain their original revision/message anchor.

The companion first drains durable queued records, then streams one source at a time. A per-source metadata checkpoint is used only when upstream provides a transcript revision; unchanged captures do not repeatedly upload. Before acknowledging content, the companion rechecks that upstream revision. A changed source retries later. Explicit authorization denial halts collection; genuine outages can use the acknowledged cached policy. Deletion returns a distinct suppression receipt, retained as a local source hash, so unrelated work continues and deleted content stays deleted. Manual queue discard also clears collection checkpoints.

Activity/search responses contain bounded previews and metadata. Complete available text is served only by the selected capture/export endpoint; preview limits never truncate the archive. This keeps a large first import from turning every list load into a full-history download.
