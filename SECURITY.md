# Security

This is an experimental archive intended to store sensitive team work under explicit policy. Do not publish a deployment before the OIDC, TLS, backup/restore, authorization and device canary gates pass.

Report vulnerabilities privately through the hosting repository's private vulnerability reporting feature when available. If it is not enabled, contact the repository maintainers privately first; do not put exploitable details or real prompts/credentials in public issues. No dedicated security email or response SLA is promised yet.

Trust boundaries: enrolled device → central ingest; human/reader token → permissioned archive; external analysis capability → one draft; untrusted transcript → inert browser text and bounded analysis context. Full/no-redaction may intentionally store secrets embedded in prompts. Access control, encrypted transport, private storage/backups and explicit retention are necessary even when redaction is disabled.

Threats worth testing include token theft/scope confusion, CSRF/OIDC replay, forged member assignment, source/revision conflicts, partial uploads becoming visible, deletion resurrection after restore, parser-induced truncation, unsafe rendering and transcript prompt injection. The product cannot authenticate the human who used a shared native account solely from its session log.

Do not expose the synthetic demo login. Do not run analysis with unrestricted write/network tools on untrusted transcripts. Do not auto-upload diagnostic bundles to public issue trackers. Keep dependency and compatibility updates reviewed.
