# BI Insights Orchestrator

Agent template for orchestrating cross-source BI insight generation in
marketing analytics.

## What this agent does

- Pulls data from multiple sources for fixed date windows.
- Normalizes KPI definitions across sources.
- Surfaces anomalies and insight candidates.
- Routes approved outputs into dashboard-ready reporting flows.

## Before you start

1. Install this agent package.
2. Install required skills.
3. Install required tools/MCP connectors.
4. Confirm your data sources are reachable.
5. Copy and adapt config examples in `config/`:
   - `tool-bindings.example.json`
   - `memory-profile.example.json`
   - `governance.example.json`

Install commands (Codex example):

```bash
./bin/skills-hub install \
  --module agents \
  --entry adtech/bi-insights-orchestrator@latest \
  --runtime codex

./bin/skills-hub install \
  adtech/analyst-copilot-bigquery-redshift@latest \
  --runtime codex
./bin/skills-hub install \
  adtech/dashboard-generator@latest \
  --runtime codex
./bin/skills-hub install \
  adtech/dashboard-qa-checker@latest \
  --runtime codex

./bin/skills-hub install \
  --module tools \
  --entry warehouse/bigquery-mcp-query-runner@latest \
  --runtime codex
./bin/skills-hub install \
  --module tools \
  --entry analytics/ga4-mcp-connector@latest \
  --runtime codex
```

## First run (copy/paste prompt)

```text
Use the BI Insights Orchestrator for:
- Current window: 2026-02-23 to 2026-03-01
- Previous window: 2026-02-16 to 2026-02-22
- Sources: GA4, BigQuery, Meta Ads

Required output:
1) Normalized KPI table (CTR, CPC, CPA, ROAS, CVR)
2) Top anomalies with evidence
3) Publish recommendation (approved or blocked)
4) Risks and data limitations
```

## What good output looks like

- Includes explicit metric formulas.
- Separates observed facts from hypotheses.
- Shows evidence values for anomalies.
- Includes a clear publish recommendation.

## Beginner safety checklist

- Start read-only for all data tools.
- Ask the agent to show limitations if any source fails.
- Require QA approval before any publish recommendation.

## Production preflight command

```bash
./bin/skills-hub run-agent \
  --agent adtech/bi-insights-orchestrator \
  --bindings agents/adtech/bi-insights-orchestrator/config/\
tool-bindings.example.json \
  --memory agents/adtech/bi-insights-orchestrator/config/\
memory-profile.example.json \
  --governance agents/adtech/bi-insights-orchestrator/config/\
governance.example.json \
  --approve-live \
  --audit-log ./tmp/bi-insights-orchestrator-run.json
```
