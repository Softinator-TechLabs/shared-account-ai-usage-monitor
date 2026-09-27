# Self-host the workspace

Run the team workspace on your own infrastructure before connecting any employee device. It is a Go service with a private PostgreSQL database. The public website and these docs are a separate static service; you do not need to deploy them for your own team.

## What you need

- A Linux host with Docker Compose or Dokploy, a persistent volume and database backups.
- Your own HTTPS domain, such as `usage.example.com`, pointing to that host. Only the app service goes behind the reverse proxy; PostgreSQL remains private.
- An owner email address and a written collection policy. Full session content can contain secrets, so decide visibility and retention with your team before enrollment.

Native Codex, Claude Code and Antigravity subscriptions continue to run in their official clients. This server does not proxy model requests or require a provider inference API key.

## 1. Prepare the source and settings

Clone the [public repository](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor). Copy `deploy/.env.example` to a private `deploy/.env` file. Set at least:

| Setting             | What to enter                                                                                                                                      |
| ------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `POSTGRES_PASSWORD` | A unique URL-safe secret; `openssl rand -hex 32` can generate one.                                                                                 |
| `PUBLIC_ORIGIN`     | Your exact HTTPS origin, for example `https://usage.example.com`, with no trailing slash.                                                          |
| `OWNER_EMAIL`       | The first owner's email address.                                                                                                                   |
| `INITIAL_POLICY`    | The collection, redaction, visibility and retention policy your team agreed to. Review the example; do not assume its full/team defaults suit you. |

The default `AUTH_MODE=password` uses one-use setup links and does not send email. For OIDC or optional viewer-key encryption, follow the [complete deployment guide](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/blob/main/docs/deployment.md). Keep `.env`, account credentials, device files and transcripts out of Git.

## 2. Deploy with Dokploy or Compose

**Dokploy:** Create a new project and a Compose service from the public Git repository. Set the Compose path to `./deploy/compose.yml`. Enter your private environment variables in Dokploy, then route your HTTPS domain to service `app` on port `8090`. Do not expose service `db`. Deploy and check `https://YOUR-DOMAIN/healthz` for an OK response.

**Docker Compose:** From the repository root, with your private `deploy/.env` ready, run:

```sh
docker compose --env-file deploy/.env -f deploy/compose.yml up -d --build
```

This Compose file deliberately publishes **no host port**. Attach an HTTPS reverse proxy to the app's Docker network and send traffic to `app:8090`; do not publish the database. A plain `docker compose up` without that proxy will not create a public login URL. The [deployment guide](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/blob/main/docs/deployment.md) covers proxy limits, backups and restoration.

## 3. Create the first owner

On first start, the app writes a 24-hour, one-use owner setup URL to `/run/setup/owner-link.txt` **inside the app container**. The runtime image has no shell. From a host with Docker administration access, find the app container and copy this file out with `docker cp`; keep the copied file private. For example:

```sh
docker compose --env-file deploy/.env -f deploy/compose.yml ps -q app
mkdir -p ~/.config/shared-ai-usage
docker cp APP_CONTAINER_ID:/run/setup/owner-link.txt ~/.config/shared-ai-usage/owner-link.txt
chmod 600 ~/.config/shared-ai-usage/owner-link.txt
```

Open the URL from that file privately and set your password. The fragment token is single-use and is not an API key. Once the owner exists, sign in, add people in **People**, choose their roles and share their one-use setup links privately. [Connect each device](/guide/connect-device) only after the employee has seen and accepted the workspace policy.

## 4. Verify and maintain

Check login/logout, an invited member, device connection, one captured session and the coverage indicators before wider enrollment. Back up the PostgreSQL volume and test restoring it. On upgrades, pin a tested release, run the integration checks and canary one device before rolling out to the team. A successful container build does not prove a Mac/Windows/Linux collector has captured everything.

The workspace is currently a single-server preview. See [current support](/guide/current-support) for platform evidence, [usage and limits](/guide/usage-and-limits) for quota interpretation, and the repository's [full deployment guide](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/blob/main/docs/deployment.md) for OIDC, deletion-ledger recovery and production details.
