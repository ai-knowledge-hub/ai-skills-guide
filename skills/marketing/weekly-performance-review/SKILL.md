---
name: weekly-performance-review
description: Analyze paid media campaign performance and recommend optimization actions. Use for ROAS, CPA, CTR trend reviews and weekly reporting.
---

# Weekly Performance Review

## When to use
Use when the user asks for ad performance analysis, weekly summaries, period-over-period comparison, or optimization recommendations.

## Inputs required
- account_ids
- date_range (default: last 7 days)
- channels (google_ads, meta_ads, tiktok_ads)

## Workflow
1. Fetch campaign metrics for the requested window and previous comparable window.
2. Compute CTR, CPC, CPA, CVR, and ROAS.
3. Flag significant shifts and anomalies.
4. Summarize winners, underperformers, and prioritized actions.

## Output format
- Executive summary
- KPI table with period deltas
- Key insights (3-5 bullets)
- Recommended actions (prioritized)

## Guardrails
- Never fabricate numbers.
- If data is missing, state limitations explicitly.
- If tool calls fail, request a CSV export from the user.
