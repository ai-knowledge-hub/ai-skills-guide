---
name: dashboard-qa-checker
description: Run pre-publish QA checks for performance marketing dashboards, including freshness, completeness, reconciliation, anomaly thresholds, and schema drift. Use when validating BI-ready data or deciding whether to block dashboard publication and send incident alerts.
---

# Dashboard QA Checker

## When to use
- Use before publishing scheduled or ad hoc dashboard updates.
- Use when validating whether source totals reconcile with dashboard aggregates.
- Use when teams need explicit pass/fail checks and block-publish behavior.

## Inputs required
- source_totals by platform and date.
- dashboard_dataset totals and field map.
- check configuration with thresholds and criticality by rule.
- latest data timestamps by source.

## Workflow
1. Run freshness check against expected ingest SLA.
2. Run completeness check for required channels and campaigns.
3. Run reconciliation check comparing spend and conversions to source totals.
4. Run anomaly checks for unexpected period-over-period deltas.
5. Run schema drift checks for added, removed, or renamed fields.
6. Assign pass, warning, or fail for each check and aggregate publish status.
7. If any critical check fails, set publish_status to blocked and emit alert payload.

## Output format
- QA Summary: overall status, failed checks count, warnings count.
- Check Results Table: check_name, severity, status, evidence, suggested fix.
- Blocking Reasons: list of critical failures and impacted outputs.
- Alert Payload: channel, title, concise incident text, run identifier.
- Publish Decision: approved or blocked.

## Guardrails
- Do not silently downgrade critical failures.
- Do not pass reconciliation if tolerance is exceeded.
- Include evidence values for every failed rule.
- Require explicit user override before recommending publish on critical fail.
