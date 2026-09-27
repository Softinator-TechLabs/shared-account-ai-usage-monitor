# Self-host

The workspace is a Go service backed by PostgreSQL. The public website and these docs are a separate static service; you may host either or both. No provider API gateway is required for native subscription clients.

1. Clone the [public repository](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor).
2. Review [`deploy/.env.example`](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/blob/main/deploy/.env.example) and choose your initial collection policy, HTTPS origin, owner email and strong secrets.
3. Deploy [`deploy/compose.yml`](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/blob/main/deploy/compose.yml) on your Docker/Compose host. Keep PostgreSQL private and back up its volume.
4. Serve the workspace behind HTTPS. Open the owner setup link privately, create users and connect devices.

The complete [deployment and OIDC guide](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/blob/main/docs/deployment.md) covers production configuration. Never copy the Softinator team's domain, credentials or employee assignments into your installation.

To host the public website separately, use `public-site/compose.yml` and route your chosen domain to its `website` service on port `8080`. The site has no connection to private telemetry and can be deployed without a database.
