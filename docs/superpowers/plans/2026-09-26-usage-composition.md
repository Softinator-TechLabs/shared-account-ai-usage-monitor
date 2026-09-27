# Usage composition and conditional quota allocation

User-approved direction: explain the method publicly and implement employee self-service model/effort/project shares, prompt counts and generated-line evidence. Extend the existing analytics flow; keep AgentsView as the reader. Execute inline with independently scoped collector/UI work and a final review.

## Contract

- Today means midnight Asia/Kolkata to request time; rolling periods retain their existing behavior.
- Per-client composition uses versioned model/token rate weights. Codex credits and Claude API-equivalent weights are never summed across providers or presented as money spent. Missing rates/counters are unweighted, not zero. Effort is observed metadata, never an invented multiplier.
- Message metadata from public AgentsView endpoints enriches usage ordinals with effort. Timestamped human user messages count as prompts; system/tool/automated messages do not. Proposed generated lines count recognized structured write/edit/patch inputs, never accepted Git lines. Missing capture stays unknown. Raw text is projected locally and not uploaded by analytics.
- Quota snapshots remain account-wide. Only closely bracketed profile identity observations can conditionally associate a token point with an account. No historical backfill from a current login. Resets, drops, changed plans, duplicate device observations, ambiguous profiles, gaps and incomplete interval boundaries must never become confident consumption.
- Conditional allocation distributes an observed same-cycle increase among the captured, weighted activity associated with that account, explicitly assuming it represents the interval. It does not prove coverage of unobserved devices or other provider surfaces. Missing/ambiguous intervals remain unallocated; no denominator means no quota percent.
- Public docs distinguish observed percentage points, conditional estimated percentage points and share of captured weighted usage. Separate windows/providers are not added into one allowance percentage. API/MCP carry the same scope and limitations as UI.

## Work

- [x] Enrich collector activity/effort with revision and pagination guards; synthetic tests.
- [x] Add rate weights, Today bounds, composition groups and conservative quota allocation; boundary/privacy tests.
- [x] Add accessible per-client pie charts, model/effort and project details, prompt/proposed-line counts and estimate states; browser tests.
- [x] Document rates, assumptions, calibration limitations, privacy and receipt boundaries.
- [x] Full checks, fresh review, PR/CI, deployment and authenticated real-data verification. Upgrade collector separately; never claim a server deploy upgrades devices.

Delivery receipts: see [status](../../status.md#usage-composition-increment). Release and owner-Mac/production smoke are verified; historical enrichment and real quota attribution coverage are not certified complete.
