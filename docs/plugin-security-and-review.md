# Plugin Security and Review

## Why plugins need extra scrutiny

Plugins package multiple moving parts together:
- skills
- agents
- tools/MCP references
- hooks
- runtime setup guidance

That makes them higher leverage than a standalone skill and therefore a higher
review surface.

## Required review questions

1. Does the plugin request only the secrets it actually needs?
2. Are write-capable tools or hooks clearly disclosed?
3. Are approval gates explicit for risky actions?
4. Do included components point to trusted registry entries?
5. Are hook names and automation behavior understandable from the README?

## Current trust model

Plugins in this repo should be treated as one of:
- `experimental`
- `reviewed`
- `deprecated`

`security_reviewed: true` should only be used when the packaged composition,
not just the individual pieces, has been reviewed.

## First-wave guardrails

- prefer read-oriented or review-oriented workflows
- disclose all required secrets
- disclose all approval expectations
- avoid implying that remote MCP connectors are trusted by default
- keep runtime manifests and setup instructions auditable in plain text
