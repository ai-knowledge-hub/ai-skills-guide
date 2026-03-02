# Meta Ads MCP Connector

## Purpose
Provide read-only Meta Ads performance retrieval via MCP.

## Capabilities
- Pull campaign/ad set metrics by date range.
- Fetch spend, impressions, clicks, conversions, and purchase value.
- Return normalized fields aligned to BI schemas.

## Guardrails
- Restrict tool to read-only reporting endpoints.
- Fail fast on missing account scope.
- Preserve source timestamps for freshness checks.
