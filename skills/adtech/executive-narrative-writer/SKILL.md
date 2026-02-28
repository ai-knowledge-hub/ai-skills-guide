---
name: executive-narrative-writer
description: Convert validated performance dashboard outputs into concise executive narratives with business impact framing, drivers, risks, and next actions. Use when stakeholders need leadership-ready summaries from BI outputs without changing underlying metrics.
---

# Executive Narrative Writer

## When to use
- Use after QA-approved dashboard datasets are available.
- Use for weekly or monthly leadership updates.
- Use when teams need concise explanation of what changed, why it changed, and what to do next.

## Inputs required
- qa_approved_dashboard_output.
- top KPI deltas and anomaly findings.
- business context: goals, budget constraints, and priority channels.
- optional audience profile: CMO, performance lead, finance partner.

## Workflow
1. Extract largest positive and negative KPI movements.
2. Attribute movements to channel, campaign, or efficiency drivers where evidence exists.
3. Separate observed facts from hypotheses.
4. Convert findings into a short executive summary with ranked recommendations.
5. Add risks and required decisions with owners and timing.

## Output format
- Headline: one-sentence performance direction.
- Executive Summary: 4 to 6 bullets with material shifts only.
- Driver Breakdown: winners, underperformers, and quantified impact.
- Risks and Unknowns: data gaps, confidence notes, and dependency risks.
- Next Actions: prioritized actions with owner, expected impact, and target date.

## Guardrails
- Never rewrite or reinterpret metric definitions.
- Avoid causal claims unless supported by data evidence.
- Keep language concise and decision-oriented.
- Explicitly mark assumptions and unresolved questions.
