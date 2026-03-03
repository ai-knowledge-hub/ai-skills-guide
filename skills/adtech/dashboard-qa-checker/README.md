# Dashboard QA Checker

Use this skill to run pre-publish QA checks and decide whether dashboard
publishing should be approved or blocked.

## What this skill does

- Checks freshness, completeness, reconciliation, anomalies, and drift.
- Returns pass/warn/fail evidence by rule.
- Produces blocking reasons and alert payloads.

## Before you start

1. Gather source totals and dashboard totals.
2. Set thresholds and criticality rules.
3. Confirm expected data freshness SLA.

## Install

```bash
./bin/skills-hub install \
  adtech/dashboard-qa-checker@latest \
  --runtime codex
```

## First run prompt

```text
Use Dashboard QA Checker for run weekly-2026-03-01.
Apply checks: freshness, completeness, reconciliation,
anomaly detection, schema drift.
Return publish decision and alert payload if blocked.
```

## What good output looks like

- Each failed check includes evidence values.
- Critical blockers are clearly separated from warnings.
- Publish decision is explicit.

## Beginner safety checklist

- Do not downgrade critical failures.
- Do not publish on critical reconciliation mismatch.
- Require explicit override to proceed after block.
