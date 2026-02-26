---
name: ai-output-eval-scorecard
description: Evaluate and score AI marketing outputs with a consistent rubric for quality, compliance, and deployment readiness.
---

# AI Output Eval Scorecard

## When to use
Use before shipping AI-generated copy, reports, or recommendations when you need a repeatable quality and risk score.

## Inputs required
- task_type
- prompt_used
- model_output
- brand_rules
- policy_rules
- scoring_weights

## Workflow
1. Parse task context and output.
2. Score dimensions: accuracy, clarity, brand fit, policy compliance, actionability.
3. Flag critical findings with severity and evidence.
4. Provide targeted rewrite recommendations.
5. Return verdict and confidence.

## Output format
- overall_score (0-100)
- verdict (`pass`, `revise`, `fail`)
- dimension_scores
- critical_findings
- rewrite_recommendations
- confidence

## Guardrails
- Never infer policy pass if rules are incomplete.
- Cite concrete evidence for each high-severity finding.
- Do not fabricate legal or platform rules.
