---
name: analyst-copilot-bigquery-redshift
description: Assist analysts with BigQuery/Redshift exploration, query drafting, metric validation, and stakeholder-ready summaries.
---

# Analyst Co-pilot on BigQuery/Redshift

## When to use
Use for ad-hoc analysis requests, KPI deep-dives, anomaly investigation, and reporting support.

## Inputs required
- schema_context
- business_question
- date_range
- dialect (bigquery|redshift)

## Workflow
1. Clarify the business question and target metric.
2. Draft SQL with explicit assumptions.
3. Validate metric logic and join safety.
4. Summarize findings and caveats.

## Output format
- SQL draft
- logic notes and assumptions
- quality checks run
- stakeholder summary

## Guardrails
- Avoid destructive SQL.
- Require explicit confirmation before write operations.
- Mark confidence limits when source coverage is partial.
