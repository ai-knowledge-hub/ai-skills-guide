# GA4 MCP Connector

Use this tool package to query GA4 metrics through MCP in a safe,
structured format.

## Current status

- This package is currently a tool specification and usage contract.
- It does **not** yet include a production Python implementation in
  `scripts/` for live execution.
- It does **not** ship a managed MCP server binary in this repository.
- You need to connect it to your own GA4 MCP server/runtime integration.

## What this tool does

- Pulls GA4 sessions, conversions, and revenue by dimensions.
- Supports fixed date windows for period comparison.
- Returns normalized output for skills and agents.

## Before you start

1. Confirm GA4 property ID.
2. Confirm MCP server is configured and authenticated.
3. Keep access read-only.

## Install

```bash
./bin/skills-hub install \
  --module tools \
  --entry analytics/ga4-mcp-connector@latest \
  --runtime codex
```

## First run prompt

```text
Use GA4 MCP Connector.
Property ID: 123456789
Date range: 2026-02-23 to 2026-03-01
Dimensions: sessionDefaultChannelGroup
Metrics: sessions, conversions, totalRevenue
Return normalized rows for dashboard input.
```

## What good output looks like

- Output rows include channel and metric values.
- Date window is applied correctly.
- Errors are explicit for invalid property/auth.

## Beginner safety checklist

- Do not use write/update endpoints.
- Validate metric/dimension allowlist first.
- Log source timestamp for freshness QA.
