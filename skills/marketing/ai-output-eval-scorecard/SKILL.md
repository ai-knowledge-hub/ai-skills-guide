---
name: ai-output-eval-scorecard
description: Evaluate and score AI marketing outputs with a consistent rubric for quality, compliance, deployment readiness, and creative operating system fit.
---

# AI Output Eval Scorecard

## When to use
Use before shipping AI-generated copy, reports, recommendations, campaign concepts, creator briefs, or creative strategy outputs when you need a repeatable quality and risk score.

## Inputs required
- task_type
- prompt_used
- model_output
- brand_rules
- policy_rules
- scoring_weights
- optional_creative_operating_system_dimensions

## Workflow
1. Parse task context and output.
2. Score core dimensions: accuracy, clarity, brand fit, policy compliance, actionability.
3. If the output is campaign, creator, cultural, or Cannes-style creative work, also score Creative Operating System dimensions:
   - utility: does it help people do something useful?
   - product_as_media_potential: can the idea live in product, service, commerce, CRM, app, packaging, or another owned surface?
   - cultural_timing: is the timing earned, safe, and relevant?
   - human_meaning: does it connect to a real behavior, tension, need, or social context?
   - brand_memory: does it use approved brand knowledge and avoid generic category language?
   - distinctiveness: would the output still be recognizable if the logo was removed?
   - creator_fit: if creators are involved, does the idea preserve creator voice and audience trust?
4. Flag critical findings with severity and evidence.
5. Provide targeted rewrite or rework recommendations.
6. Return verdict and confidence.

## Output format
- overall_score (0-100)
- verdict (`pass`, `revise`, `fail`)
- dimension_scores
- creative_operating_system_scores (when relevant)
- critical_findings
- rewrite_recommendations
- confidence

## Guardrails
- Never infer policy pass if rules are incomplete.
- Cite concrete evidence for each high-severity finding.
- Do not fabricate legal or platform rules.
- Do not reward polished language if the idea lacks evidence, utility, or brand permission.
- Flag unsupported claims even when they sound strategically attractive.
