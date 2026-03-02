# GA4 MCP Connector

## Purpose
Provide standardized GA4 query access via MCP for marketing skills and agents.

## Capabilities
- Pull sessions, users, conversions, and revenue by dimension.
- Query fixed date windows for period comparisons.
- Return normalized payloads for downstream skills.

## Guardrails
- Enforce read-only query scope.
- Validate requested dimensions/metrics against allowlist.
- Return clear errors for invalid property or auth failures.
