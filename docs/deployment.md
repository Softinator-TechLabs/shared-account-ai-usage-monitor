# Dokploy / production deployment

Production requires a chosen HTTPS origin, PostgreSQL persistence/backups and an owner email (OIDC is optional). The development demo must never be publicly proxied. No provider inference API key is needed.

1. Copy `deploy/.env.example` to a private environment. Set a URL-safe strong database password and exact `PUBLIC_ORIGIN` (no trailing slash), owner email and initial collection policy. Default `AUTH_MODE=password` uses private invitation links. For optional OIDC (`both` or `oidc`), set issuer/client credentials and register `<PUBLIC_ORIGIN>/auth/callback`; map exact verified subjects.
2. Deploy `deploy/compose.yml` in Dokploy with the product directory as source. Route your TLS domain to the `app` service port 8090 on the private application network. The example localhost port is for a local reverse proxy; adapt Dokploy's network attachment without exposing PostgreSQL. Store secrets in Dokploy, not Git. Reverse-proxy request body limit must allow at least 2 MiB for base64 chunk requests and 64 MiB for ordinary imports; idle/timeouts must permit ingestion.
3. Confirm `/healthz`, owner email login/logout, member login, invitation acknowledgement, full synthetic text ingest/search/export, and direct-ID isolation under self/managers policy. Test a revoked member. Only then enroll an agreed employee device.
4. Persist the `archive` PostgreSQL volume. Back up with PostgreSQL's tools, encrypt backups and test restoration. Export `/api/v1/deletion-ledger` with an owner session into a **separately retained current ledger** after deletions; do not rely only on an older DB backup. Restore the database with the app inaccessible, mount the current ledger and set `RESTORE_DELETION_LEDGER=/private/deletions.json` at startup. Startup replays it before serving traffic. External backups remain until their own retention expires; database deletion cannot physically erase old external media.
5. Pin verified container digests at release/deploy time. The example version tags are candidate build inputs, not proof they were deployed. Upgrade only after integration tests and one-device canary. Keep the previous image and a tested database migration/restore plan.

The server runs retention at startup and hourly. It deletes wholly expired source histories, associated comments/analysis jobs, stale staging and expired tokens. Existing policy representation is not rewritten in place when changing policy; use explicit deletion when older raw content must disappear.

Real DNS, TLS, optional OIDC tenant integration, actual Dokploy/container startup and real-device receipts are deployment gates. See status.md for what has actually been tested.

## First owner and team setup

The first start writes a 24-hour single-use owner setup URL into `/run/setup/owner-link.txt` inside the app container, mode 0600. Copy it using your administrator container access (`docker cp <app-container>:/run/setup/owner-link.txt ./owner-link.txt`, then restrict the local file). Open it privately and choose your password. The URL uses a fragment so the setup token is not sent in page requests or referrers. It is submitted only to the same-origin setup endpoint. A restart can regenerate this link only until the configured owner has a password; it never resets an established password.

For a headless deployment, optionally set `OWNER_SETUP_TOKEN` to 32 random bytes encoded as 64 hex characters (for example, `openssl rand -hex 32`) in the private deployment environment. Until the first password is set, the setup URL uses that token and remains stable across restarts. It is rejected after successful setup and can never reset an existing password. Remove the environment value after setup. Never commit the token or setup URL.

People → Add person creates an email identity with Member, Project manager or Owner role. Download its setup link and share privately. No email delivery service is required or configured. Reset login creates another one-use setup link; successful password replacement revokes browser/read sessions. Device tokens are managed separately. Login attempts are limited by account and server IP; behind a proxy the per-IP budget applies to the proxy address because arbitrary forwarded headers are deliberately not trusted.

The supplied compose uses a private tmpfs for bootstrap output. Keep the container administrator boundary private. Do not publish port 8090 directly except through your HTTPS reverse proxy, and do not expose PostgreSQL. Only one server replica is supported in this experimental release.
