# Meta/Google Weekly Performance Review

Use this skill to generate a standardized weekly narrative across Google
Ads and Meta Ads.

## What this skill does

- Compares current and previous period performance.
- Computes core paid media KPIs.
- Produces prioritized actions and risk notes.

## Before you start

1. Provide Meta and Google account IDs.
2. Provide analysis date range.
3. Confirm required channels.

## Install

```bash
./bin/skills-hub install \
  marketing/meta-google-weekly-performance-review@latest \
  --runtime codex
```

## First run prompt

```text
Use Meta/Google Weekly Performance Review.
Date range: last 7 days
Compare against prior 7 days.
Return executive summary, KPI table,
key insights, and recommended actions.
```

## What good output looks like

- KPI deltas are clear and accurate.
- Winners and underperformers are explicit.
- Limitations are called out.

## Beginner safety checklist

- Never fabricate missing numbers.
- Request exports if data access fails.
- Flag low-confidence conclusions.
