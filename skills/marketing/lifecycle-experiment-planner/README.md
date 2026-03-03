# Lifecycle Experiment Planner

Use this skill to design statistically coherent lifecycle experiments.

## What this skill does

- Validates hypothesis quality.
- Calculates sample-size guidance.
- Defines stop/continue criteria.

## Before you start

1. Provide baseline rate and MDE.
2. Set confidence and power assumptions.
3. Define primary and guardrail metrics.

## Install

```bash
./bin/skills-hub install \
  marketing/lifecycle-experiment-planner@latest \
  --runtime codex
```

## First run prompt

```text
Use Lifecycle Experiment Planner.
Channel: email journey
Baseline: 3.8% conversion
MDE: 12%
Return hypothesis, design table, sample-size guidance,
and stop/continue criteria.
```

## What good output looks like

- Hypothesis is specific and testable.
- Sample-size assumptions are explicit.
- Decision criteria are unambiguous.

## Beginner safety checklist

- Keep one primary metric.
- Avoid early stopping without guardrail breach.
- Reject vague hypotheses.
