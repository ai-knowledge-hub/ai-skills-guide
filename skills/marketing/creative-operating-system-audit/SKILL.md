---
name: creative-operating-system-audit
description: Audit a marketing team's creative operating system: brand memory, cultural context, evaluation, product-as-media surfaces, creator workflow, feedback loops, and governance.
---

# Creative Operating System Audit

## When to use
Use this skill when a team asks whether its AI-assisted marketing process can produce distinctive, useful, governed creative work rather than more generic content.

## Inputs required
- Brand or client name
- Current creative workflow
- Brand guidelines or memory sources
- Campaign examples or recent outputs
- Review and approval process
- Data and feedback sources
- Creator or partner workflow, if any
- Product, app, CRM, retail, commerce, or service surfaces

## Workflow
1. Map the current creative workflow from brief to launch to post-launch learning.
2. Score seven dimensions: brand memory, cultural context, judgment/evaluation, utility, product-as-media, creator collaboration, and governance.
3. Identify evidence for each score. Separate observed facts from assumptions.
4. Flag gaps that create AI slop risk: generic outputs, weak memory, vague evaluation, untracked rights, missing feedback loops.
5. Recommend a 30-day improvement plan with owner, first artifact, and success signal.

## Output format
Return:
- `overall_maturity_score` from 0-100
- `dimension_scores`
- `evidence_table`
- `top_gaps`
- `30_day_action_plan`
- `risks_if_ignored`
- `next_skills_to_use`

## Guardrails
- Do not treat asset volume as creative maturity.
- Do not infer governance controls without evidence.
- Call out missing source material explicitly.
- Keep recommendations practical and owned by real functions.
- Separate creative judgment gaps from tooling gaps.
