---
name: dashboard-generator
description: Build or refresh deterministic weekly performance dashboard outputs from standardized marketing tables for BI tools like Looker, Power BI, or Tableau. Use when asked to generate channel summaries, anomaly panels, action panels, or a fixed reporting layout for recurring performance reviews.
---

# Dashboard Generator

## When to use
- Use for recurring dashboard builds from cleaned performance datasets.
- Use when stakeholders require the same weekly panel order and structure.
- Use when asked to output both machine-ready dataset slices and human-readable dashboard sections.

## Inputs required
- normalized_performance_table with date, channel, campaign, spend, clicks, impressions, conversions, revenue.
- date_window_current and date_window_previous.
- output_contract (default): executive summary, channel table, anomaly panel, action panel.
- optional thresholds for highlighting changes.

## Workflow
1. Validate required columns before any transformation.
2. Aggregate by requested grain: channel first, then campaign when requested.
3. Compute canonical metrics: CTR, CPC, CVR, CPA, ROAS.
4. Compute period deltas using the same grain and metric definitions.
5. Build deterministic dashboard payload sections in fixed order.
6. Return both section content and a compact publish-ready data object.

## Output format
- Executive Summary: 3 to 5 bullets with largest business-impact changes.
- Channel Table: channel, spend, conversions, CPA, ROAS, week-over-week deltas.
- Anomaly Panel: metric, segment, observed delta, threshold, severity, likely cause hypothesis.
- Action Panel: ranked actions with expected impact and confidence.
- Publish Payload: stable JSON keys for BI layer ingestion.

## Guardrails
- Never fabricate data for missing channels or campaigns.
- Preserve metric definitions exactly as provided by upstream dictionary.
- Keep section order deterministic across runs.
- Flag low confidence when previous period volume is below threshold.
