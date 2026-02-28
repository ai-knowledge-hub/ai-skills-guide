---
name: weekly-performance-review-bi
description: Orchestrate end-to-end weekly BI performance reporting by chaining dashboard generation, QA validation, and executive narrative writing. Use when teams need a deterministic weekly workflow that blocks publish on critical data-quality failures and only produces leadership summaries after QA approval.
---

# Weekly Performance Review BI

## When to use
- Use for recurring weekly performance marketing reporting with BI publish gates.
- Use when output must follow a fixed structure across runs.
- Use when reporting must stop automatically on critical data-quality failures.

## Inputs required
- normalized_performance_table with standardized marketing metrics.
- source_totals and data timestamps for QA.
- metric_dictionary and QA rule thresholds.
- audience profile for executive summary tone.
- reporting windows: current period and comparison period.

## Workflow
1. Run `dashboard-generator` on normalized data.
2. Validate `dashboard-generator` output schema and required sections.
3. Run `dashboard-qa-checker` against source totals, freshness, completeness, reconciliation, anomaly rules, and schema drift.
4. If QA result is `blocked`, stop workflow and return QA package plus alert payload.
5. If QA result is `approved`, run `executive-narrative-writer` with QA-approved dashboard output.
6. Return consolidated package with publish decision, dashboard sections, and executive narrative.

## Output format
- Run Metadata: run_id, reporting windows, execution timestamp.
- Publish Decision: approved or blocked with reasons.
- Dashboard Package: executive summary, channel table, anomaly panel, action panel, publish payload.
- QA Package: summary, check results, blocking reasons, alert payload.
- Executive Narrative: headline, concise summary, driver breakdown, risks, next actions.

## Guardrails
- Never run executive narrative step when QA status is blocked.
- Never override a critical QA failure without explicit user approval.
- Preserve metric definitions and output schema contracts across all steps.
- Keep section order stable so BI consumers receive deterministic outputs.
