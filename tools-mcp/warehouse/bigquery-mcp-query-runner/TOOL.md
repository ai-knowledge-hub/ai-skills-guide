# BigQuery MCP Query Runner

## Purpose
Run parameterized warehouse queries through MCP for analytics workflows.

## Capabilities
- Execute parameterized SQL templates.
- Enforce read-only query patterns.
- Return typed rows for BI transformations.

## Guardrails
- Reject non-read-only SQL statements.
- Enforce query timeout and row limits.
- Surface query diagnostics in failures.
