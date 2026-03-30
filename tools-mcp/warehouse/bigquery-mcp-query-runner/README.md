# BigQuery MCP Query Runner

Use this tool package to run parameterized, read-only BigQuery queries
through MCP.

## Current status

- This package is currently a tool specification and usage contract.
- It does **not** yet include a production Python implementation in
  `scripts/` for live execution.
- It does **not** ship a managed MCP server binary in this repository.
- You need to connect it to your own BigQuery MCP server/runtime
  integration.

## What this tool does

- Executes parameterized SQL templates.
- Enforces read-only query behavior.
- Returns typed rows for analytics skills and agents.

## Operational metadata

- Connected system: Google BigQuery
- Access level: read-only
- Trust boundary: remote MCP server
- Auth required: authenticated dataset access in your MCP runtime
- Approval boundary: safe for parameterized read-only queries; require human approval before broadening dataset scope or enabling any non-read-only SQL path

## Before you start

1. Confirm dataset/table access.
2. Prepare parameterized SQL template.
3. Set query timeout and row limits.

## Install

```bash
./bin/skills-hub install \
  --module tools \
  --entry warehouse/bigquery-mcp-query-runner@latest \
  --runtime codex
```

## First run prompt

```text
Use BigQuery MCP Query Runner.
SQL template:
SELECT channel, SUM(spend) AS spend
FROM marketing.performance
WHERE date BETWEEN @start AND @end
GROUP BY 1
Params: start=2026-02-23, end=2026-03-01
Return typed rows and row_count.
```

## What good output looks like

- Query executes with provided parameters.
- Output schema is stable and typed.
- Non-read-only SQL is rejected with clear diagnostics.

## Beginner safety checklist

- Allow only SELECT-style queries.
- Enforce timeout and row limits.
- Capture and return query diagnostics on failure.
