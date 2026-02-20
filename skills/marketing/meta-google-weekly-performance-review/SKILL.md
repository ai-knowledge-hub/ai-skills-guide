---
name: meta-google-weekly-performance-review
description: Analyze weekly paid media performance across Meta and Google Ads with ROAS/CPA/CTR trend explanation and prioritized actions.
---

# Meta/Google Weekly Performance Review

## When to use
Use for weekly reviews, trend analysis, budget reallocation decisions, and "what changed" questions.

## Inputs required
- google_account_id
- meta_account_id
- date_range (default last 7 days)

## Workflow
1. Pull campaign metrics for current and previous period.
2. Calculate CTR, CPC, CVR, CPA, and ROAS.
3. Identify top/bottom campaigns and major deltas.
4. Produce a stakeholder-ready narrative.

## Output format
- Executive Summary
- KPI Table (with period deltas)
- Key Insights
- Recommended Actions
- Risks / Data Limitations

## Guardrails
- Never fabricate numbers.
- If data is unavailable, request export files.
- Flag low confidence when conversion volume is low.
