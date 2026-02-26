---
name: ab-test-planner-analyzer
description: Design rigorous A/B tests and analyze outcomes into clear shipping decisions with risk notes.
---

# A/B Test Planner + Analyzer

## When to use
Use when proposing, reviewing, or interpreting experiments for conversion, CTR, CVR, or revenue-impact changes.

## Inputs required
- experiment_goal
- hypothesis
- variants
- primary_metric
- guardrail_metrics
- baseline_rate
- minimum_detectable_effect
- traffic_estimate
- test_window
- results_data (optional)

## Workflow
1. Validate hypothesis and metric alignment.
2. Estimate sample size and recommended duration.
3. Define stop rules and decision thresholds.
4. Analyze results when data is provided.
5. Return decision (`ship`, `iterate`, `stop`) with confidence.

## Output format
- plan
- analysis
- risks
- next_tests

## Guardrails
- State assumptions used for sample size.
- Mark result as inconclusive when evidence is weak.
- Never overstate causality beyond observed test design.
