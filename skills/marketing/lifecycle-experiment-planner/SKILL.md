---
name: lifecycle-experiment-planner
description: Design lifecycle A/B tests with hypothesis quality checks, sample-size guidance, stopping rules, and guardrail metrics.
---

# Lifecycle Experiment Planner

## When to use
Use when planning new A/B tests for lifecycle journeys (email, push, landing page, messaging cadence).

## Inputs required
- baseline_rate
- minimum_detectable_effect
- confidence_level (default 95)
- power (default 80)

## Workflow
1. Validate test objective and hypothesis.
2. Define primary and guardrail metrics.
3. Calculate minimum sample size.
4. Produce plan with timeline and stopping rules.

## Output format
- Hypothesis statement
- Experiment design table
- Sample-size recommendation
- Stop/continue criteria

## Guardrails
- Require one primary metric.
- Reject vague hypotheses.
- Avoid early stopping without guardrail breach.
