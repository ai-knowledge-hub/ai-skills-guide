# Lifecycle Journey Trigger Designer

Use this skill to design event-based lifecycle triggers, sequencing,
and measurement plans.

## What this skill does

- Defines entry/progression/exit triggers.
- Designs channel timing and suppression logic.
- Returns rollout and measurement plans.

## Before you start

1. Define journey goal and audience.
2. Provide trigger events and channel constraints.
3. Define success metrics.

## Install

```bash
./bin/skills-hub install \
  marketing/lifecycle-journey-trigger-designer@latest \
  --runtime codex
```

## First run prompt

```text
Use Lifecycle Journey Trigger Designer.
Goal: increase activation in first 14 days.
Audience: new signups
Return journey blueprint, trigger rules,
channel sequence, suppression rules,
and measurement plan.
```

## What good output looks like

- Trigger logic is unambiguous.
- Suppression/frequency controls are explicit.
- Measurement plan ties to goal.

## Beginner safety checklist

- Prevent overlapping trigger collisions.
- Respect legal communication constraints.
- Flag missing event instrumentation.
