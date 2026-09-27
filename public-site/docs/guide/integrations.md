# Integrations and upstreams

The product observes native client output and joins it with the team's own identities, projects and account readings. A local AgentsView installation currently supplies part of the session-reading experience. AgentsView is an independent open-source project, versioned and installed separately; it is not the identity, policy or quota authority of this product.

The integration boundary is deliberate: a source adapter can evolve or be replaced without changing how the workspace represents people, accounts, evidence coverage and permissions. This is an architectural direction, not a claim that another reader is already implemented. Consult the [architecture](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/blob/main/docs/architecture.md), [compatibility matrix](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/blob/main/compatibility/support-matrix.json) and [third-party notices](https://github.com/Softinator-TechLabs/shared-account-ai-usage-monitor/blob/main/THIRD_PARTY_NOTICES.md) for the current implementation and licenses.

The read-only analytics MCP uses the same permissioned evidence boundaries as the web app. Imported prompt text is data, never an instruction to an agent.
