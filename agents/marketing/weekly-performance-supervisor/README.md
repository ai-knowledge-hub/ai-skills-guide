# Weekly Performance Supervisor

Agent template for deterministic weekly performance reporting with
QA-gated publish control.

## What this agent does

- Orchestrates weekly reporting in a fixed sequence.
- Builds dashboard sections from normalized data.
- Runs QA checks before publish.
- Generates executive narrative only when QA passes.

## Before you start

1. Install this agent package.
2. Install required skills.
3. Install required tools/MCP connectors.
4. Confirm date windows and required channels.
5. Copy and adapt config examples in `config/`:
   - `tool-bindings.example.json`
   - `memory-profile.example.json`
   - `governance.example.json`

Install commands (Codex example):

```bash
./bin/skills-hub install \
  --module agents \
  --entry marketing/weekly-performance-supervisor@latest \
  --runtime codex

./bin/skills-hub install \
  adtech/dashboard-generator@latest \
  --runtime codex
./bin/skills-hub install \
  adtech/dashboard-qa-checker@latest \
  --runtime codex
./bin/skills-hub install \
  adtech/executive-narrative-writer@latest \
  --runtime codex

./bin/skills-hub install \
  --module tools \
  --entry analytics/ga4-mcp-connector@latest \
  --runtime codex
./bin/skills-hub install \
  --module tools \
  --entry warehouse/bigquery-mcp-query-runner@latest \
  --runtime codex
```

## First run (copy/paste prompt)

```text
Use Weekly Performance Supervisor for:
- Current period: 2026-02-23 to 2026-03-01
- Previous period: 2026-02-16 to 2026-02-22
- Required channels: Paid Search, Paid Social
- Audience: CMO

Required output in this order:
1) Publish decision
2) Dashboard package
3) QA package
4) Executive narrative (only if approved)
```

## What good output looks like

- Section order is deterministic.
- QA result is explicit and evidence-backed.
- Publish is blocked on critical QA failure.
- Narrative appears only when publish decision is approved.

## Beginner safety checklist

- Keep reconciliation and freshness checks enabled.
- Never force publish when critical QA fails.
- Ask for missing-source limitations in the final output.

## Production preflight command

```bash
./bin/skills-hub run-agent \
  --agent marketing/weekly-performance-supervisor \
  --bindings agents/marketing/weekly-performance-supervisor/config/\
tool-bindings.example.json \
  --memory agents/marketing/weekly-performance-supervisor/config/\
memory-profile.example.json \
  --governance agents/marketing/weekly-performance-supervisor/config/\
governance.example.json \
  --approve-live \
  --audit-log ./tmp/weekly-performance-supervisor-run.json
```
