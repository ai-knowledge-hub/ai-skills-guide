# Cross-Channel Budget Pacing Agent

Use this skill to monitor spend pacing and propose bounded reallocations
across channels.

## What this skill does

- Compares actual vs plan.
- Detects pacing and efficiency anomalies.
- Recommends constrained budget shifts.

## Before you start

1. Provide channel performance data.
2. Provide targets and constraints.
3. Provide calendar context.

## Install

```bash
./bin/skills-hub install \
  marketing/cross-channel-budget-pacing-agent@latest \
  --runtime codex
```

## First run prompt

```text
Use Cross-Channel Budget Pacing Agent.
Date range: last 7 days
Return pacing status, channel summary,
anomalies, and next 7-day reallocation plan.
Respect max 15% shift per channel.
```

## What good output looks like

- Variance vs plan is explicit.
- Constraints are respected.
- Actions are prioritized and bounded.

## Beginner safety checklist

- Never exceed budget constraints.
- Separate facts from speculation.
- Highlight missing channel data.
