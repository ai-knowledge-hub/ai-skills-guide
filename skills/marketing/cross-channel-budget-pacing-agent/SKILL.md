---
name: cross-channel-budget-pacing-agent
description: Track spend pacing and performance by channel, detect anomalies, and propose bounded reallocations.
---

# Cross-Channel Budget Pacing Agent

## When to use
Use for weekly and intra-week budget management when spend and efficiency need active pacing control.

## Inputs required
- date_range
- channel_data
- targets
- constraints
- calendar_context

## Workflow
1. Compare actual vs planned spend by channel and campaign.
2. Calculate variance and efficiency metrics (CPA, ROAS where available).
3. Detect anomalies and classify severity.
4. Propose constrained budget shifts.
5. Return next-7-day action plan.

## Output format
- pacing_status
- channel_summary
- anomalies
- reallocation_plan
- next_7d_actions

## Guardrails
- Do not recommend budget moves beyond declared constraints.
- Separate observed anomalies from speculative causes.
- Surface missing channel data explicitly.
