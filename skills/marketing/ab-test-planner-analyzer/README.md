# A/B Test Planner + Analyzer

Use this skill to design strong experiments and convert results into
clear decisions.

## What this skill does

- Validates hypotheses and metrics.
- Recommends sample size and duration.
- Analyzes results into `ship`, `iterate`, or `stop`.

## Before you start

1. Define experiment goal and primary metric.
2. Provide baseline rate and MDE.
3. Provide traffic estimate and test window.

## Install

```bash
./bin/skills-hub install \
  marketing/ab-test-planner-analyzer@latest \
  --runtime codex
```

## First run prompt

```text
Use A/B Test Planner + Analyzer.
Goal: improve trial signup rate.
Primary metric: signup CVR
Baseline: 4.2%
MDE: 10%
Return sample size, stop rules, and decision framework.
```

## What good output looks like

- Assumptions are explicit.
- Primary and guardrail metrics are clear.
- Decision is tied to evidence quality.

## Beginner safety checklist

- Use one primary metric.
- Treat weak evidence as inconclusive.
- Avoid causal claims beyond test design.
