# Analyst Co-pilot on BigQuery/Redshift

Use this skill to turn analyst questions into safe SQL drafts and
stakeholder-ready findings.

## What this skill does

- Clarifies business questions and target metrics.
- Drafts SQL for BigQuery or Redshift.
- Validates metric logic and join safety.
- Produces concise findings with caveats.

## Before you start

1. Confirm schema context and date range.
2. Decide SQL dialect: `bigquery` or `redshift`.
3. Start in read-only mode.

## Install

```bash
./bin/skills-hub install \
  adtech/analyst-copilot-bigquery-redshift@latest \
  --runtime codex
```

## First run prompt

```text
Use Analyst Co-pilot on BigQuery/Redshift.
Business question: Why did conversion rate drop last week?
Dialect: bigquery
Date range: 2026-02-23 to 2026-03-01
Return SQL draft, logic assumptions, and a stakeholder summary.
```

## What good output looks like

- SQL is syntactically valid for selected dialect.
- Assumptions are explicit.
- Caveats are clearly stated.
- No destructive operations.

## Beginner safety checklist

- Do not run write queries.
- Verify table names before execution.
- Ask for missing schema fields explicitly.
