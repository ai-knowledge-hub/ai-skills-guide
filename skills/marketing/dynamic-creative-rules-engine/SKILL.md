---
name: dynamic-creative-rules-engine
description: Build dynamic creative personalization rules based on audience signals with policy and brand guardrails.
---

# Dynamic Creative Rules Engine

## When to use
Use for personalized ad or lifecycle creative systems where message components need to vary by segment and context.

## Inputs required
- audience_segments
- context_signals
- creative_components
- brand_constraints
- policy_constraints
- objective_metric

## Workflow
1. Map segment and context signal hierarchy.
2. Define creative component matrix (headline, body, CTA, visual hook).
3. Apply brand and policy constraints to each combination.
4. Rank candidate assemblies by objective fit.
5. Return deployable rules and test matrix.

## Output format
- segmentation_logic
- creative_rules
- blocked_combinations
- test_matrix
- monitoring_metrics

## Guardrails
- Never allow combinations that violate policy constraints.
- Enforce hard brand constraints before optimization.
- Keep a human-review checkpoint for high-risk segments.
