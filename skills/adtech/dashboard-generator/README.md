# Dashboard Generator

Use this skill to build deterministic weekly dashboard sections from a
normalized performance table.

## What this skill does

- Computes canonical KPIs (CTR, CPC, CVR, CPA, ROAS).
- Builds fixed output sections in stable order.
- Produces a publish-ready payload for BI tools.

## Before you start

1. Ensure required columns exist in your normalized table.
2. Set current and previous date windows.
3. Confirm anomaly thresholds (optional).

## Install

```bash
./bin/skills-hub install \
  adtech/dashboard-generator@latest \
  --runtime codex
```

## First run prompt

```text
Use Dashboard Generator.
Current window: 2026-02-23 to 2026-03-01
Previous window: 2026-02-16 to 2026-02-22
Output sections in this order:
1) Executive Summary
2) Channel Table
3) Anomaly Panel
4) Action Panel
```

## What good output looks like

- Section ordering is deterministic.
- KPI formulas are consistent.
- Missing data is explicitly flagged.

## Beginner safety checklist

- Never invent missing channels.
- Keep metric definitions unchanged.
- Preserve output schema for weekly comparability.
