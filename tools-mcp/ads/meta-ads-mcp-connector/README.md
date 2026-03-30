# Meta Ads MCP Connector

Use this tool package to pull Meta Ads performance data via MCP for
analysis and reporting.

## Current status

- This package is currently a tool specification and usage contract.
- It does **not** yet include a production Python implementation in
  `scripts/` for live execution.
- It does **not** ship a managed MCP server binary in this repository.
- You need to connect it to your own Meta Ads MCP server/runtime
  integration.

## What this tool does

- Pulls campaign/ad set metrics for date windows.
- Returns spend, impressions, clicks, and conversions.
- Preserves source timestamps for QA checks.

## Operational metadata

- Connected system: Meta Ads
- Access level: read-only
- Trust boundary: remote MCP server
- Auth required: authenticated Meta Ads account scope in your MCP runtime
- Approval boundary: safe for reporting queries only; require human approval before pairing this connector with workflows that can modify campaigns, budgets, or creative state

## Before you start

1. Confirm Meta ad account ID scope.
2. Confirm MCP server auth is valid.
3. Keep tool access read-only.

## Install

```bash
./bin/skills-hub install \
  --module tools \
  --entry ads/meta-ads-mcp-connector@latest \
  --runtime codex
```

## First run prompt

```text
Use Meta Ads MCP Connector.
Account ID: act_1234567890
Date range: 2026-02-23 to 2026-03-01
Fields: campaign_name, spend, impressions, clicks, actions
Return normalized campaign-level performance rows.
```

## What good output looks like

- Campaign rows include requested metrics.
- Missing scope errors are clear and actionable.
- Source timestamp is included for downstream freshness checks.

## Beginner safety checklist

- Reject requests without account scope.
- Keep connector read-only.
- Do not infer missing conversion mappings silently.
