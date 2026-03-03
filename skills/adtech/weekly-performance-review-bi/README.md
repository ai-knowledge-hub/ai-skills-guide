# Weekly Performance Review BI

Use this skill to orchestrate deterministic weekly BI reporting with a
hard QA gate before publication.

## What this skill does

- Runs dashboard generation.
- Runs dashboard QA checks.
- Runs executive narrative only when QA passes.

## Before you start

1. Prepare normalized performance data.
2. Prepare source totals and freshness timestamps.
3. Define QA thresholds and audience.

## Install

```bash
./bin/skills-hub install \
  adtech/weekly-performance-review-bi@latest \
  --runtime codex
```

## First run prompt

Use the starter template:

- [examples/starter-prompt.md](examples/starter-prompt.md)

## What good output looks like

- Publish decision is explicit.
- Dashboard section order is deterministic.
- QA package includes evidence values.
- Narrative appears only when QA approved.

## Beginner safety checklist

- Never bypass critical QA failures.
- Keep metric dictionary fixed across runs.
- Require alerting on blocked publish.
