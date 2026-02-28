# Weekly Performance Review BI - Starter Prompt Template

Use `weekly-performance-review-bi` and run the full workflow in this strict order:
1. `dashboard-generator`
2. `dashboard-qa-checker`
3. `executive-narrative-writer` only if QA status is `approved`

Input contract:
- Date windows:
  - current_period: `{{CURRENT_PERIOD_START}}` to `{{CURRENT_PERIOD_END}}`
  - previous_period: `{{PREVIOUS_PERIOD_START}}` to `{{PREVIOUS_PERIOD_END}}`
- Data sources:
  - GA4 export/table: `{{GA4_SOURCE}}`
  - Ad platform export/table: `{{ADS_SOURCE}}`
  - Warehouse model/table: `{{WAREHOUSE_SOURCE}}`
- Required normalized columns:
  - `date`, `channel`, `campaign`, `spend`, `impressions`, `clicks`, `conversions`, `revenue`

Metric dictionary (do not redefine):
- CTR = clicks / impressions
- CPC = spend / clicks
- CPA = spend / conversions
- ROAS = revenue / spend
- CVR = conversions / clicks

Fixed output contract (exact section order):
1. Executive Summary
2. Channel Table
3. Anomaly Panel
4. Action Panel

QA checks (must run before publish):
- freshness:
  - fail if latest source timestamp is older than `{{FRESHNESS_SLA_HOURS}}` hours
- completeness:
  - fail if required channels are missing: `{{REQUIRED_CHANNELS_CSV}}`
- reconciliation:
  - fail if spend delta > `{{RECONCILIATION_SPEND_TOLERANCE_PCT}}%`
  - fail if conversions delta > `{{RECONCILIATION_CONV_TOLERANCE_PCT}}%`
- anomaly detection:
  - flag if week-over-week delta exceeds `{{ANOMALY_DELTA_PCT}}%`
- schema drift:
  - fail if required columns are renamed or removed

Publish rule:
- If any critical QA check fails:
  - set `publish_decision=blocked`
  - do not generate executive narrative
  - produce Slack/Teams alert payload using:
    - channel: `{{ALERT_CHANNEL}}`
    - destination: `{{ALERT_DESTINATION}}`
    - mention group: `{{ALERT_MENTION}}`
- If all critical checks pass:
  - set `publish_decision=approved`
  - generate executive narrative for audience: `{{AUDIENCE}}`

Final response format:
- Run Metadata: run_id, execution_timestamp, date windows
- Publish Decision: approved/blocked with reasons
- Dashboard Package: Executive Summary, Channel Table, Anomaly Panel, Action Panel, publish payload
- QA Package: check results table with evidence, blocking reasons, alert payload (if blocked)
- Executive Narrative: include only when publish_decision=approved

Non-negotiable guardrails:
- Never fabricate numeric values.
- Never bypass critical QA failures.
- Keep output schema deterministic across runs.
