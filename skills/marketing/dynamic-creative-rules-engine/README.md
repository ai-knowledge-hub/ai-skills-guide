# Dynamic Creative Rules Engine

Use this skill to define audience-aware creative assembly rules with
brand and policy guardrails.

## What this skill does

- Maps segment and context signal logic.
- Builds allowed creative component combinations.
- Returns deployable rules and test matrix.

## Before you start

1. Provide audience segments and context signals.
2. Provide creative components.
3. Provide brand and policy constraints.

## Install

```bash
./bin/skills-hub install \
  marketing/dynamic-creative-rules-engine@latest \
  --runtime codex
```

## First run prompt

```text
Use Dynamic Creative Rules Engine.
Segments: new visitors, returning users
Signals: channel, device, intent bucket
Return segmentation logic, creative rules,
blocked combinations, and test matrix.
```

## What good output looks like

- Rule logic is explicit and testable.
- Blocked combinations are documented.
- Monitoring metrics are defined.

## Beginner safety checklist

- Enforce hard policy constraints first.
- Keep human review for high-risk segments.
- Avoid uncontrolled variant explosion.
