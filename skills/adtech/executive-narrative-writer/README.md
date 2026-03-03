# Executive Narrative Writer

Use this skill to convert QA-approved dashboard outputs into concise,
leadership-ready narratives.

## What this skill does

- Summarizes KPI movement and drivers.
- Distinguishes facts from hypotheses.
- Produces risks, unknowns, and next actions.

## Before you start

1. Ensure QA status is approved.
2. Provide KPI deltas and anomaly findings.
3. Define audience (for example CMO or finance lead).

## Install

```bash
./bin/skills-hub install \
  adtech/executive-narrative-writer@latest \
  --runtime codex
```

## First run prompt

```text
Use Executive Narrative Writer.
Audience: CMO
Input: QA-approved dashboard output for current week.
Return headline, executive summary, drivers,
risks/unknowns, and next actions.
```

## What good output looks like

- Main conclusion appears in first lines.
- Drivers are quantified where possible.
- Assumptions are explicitly marked.

## Beginner safety checklist

- Do not rewrite metric definitions.
- Avoid causal claims without evidence.
- Keep recommendations specific and actionable.
