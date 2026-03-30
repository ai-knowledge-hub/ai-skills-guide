# Campaign QA Supervisor

Agent template for campaign QA gating with clear remediation outputs.

## What this agent does

- Runs pre-launch and in-flight QA checks for campaigns.
- Flags missing tracking fields and policy issues.
- Returns pass/warn/fail with remediation actions.
- Blocks launch recommendations on critical failures.

## Operational metadata

- Role: governance agent for campaign QA, tracking integrity, and launch readiness
- Autonomy level: execution-capable-with-approval
- Approval boundary: may assess launch readiness and draft remediation alerts; require human approval before overriding critical failures or taking live launch actions
- Outputs:
  - pass, warn, or fail decision
  - critical blocker list
  - remediation action list
  - alert summary for escalation

## Before you start

1. Install this agent package.
2. Install its required skill dependency.
3. Connect campaign config and alerting tools.
4. Copy and adapt config examples in `config/`:
   - `tool-bindings.example.json`
   - `memory-profile.example.json`
   - `governance.example.json`

Install commands (Codex example):

```bash
./bin/skills-hub install \
  --module agents \
  --entry marketing/campaign-qa-supervisor@latest \
  --runtime codex

./bin/skills-hub install \
  adtech/policy-brand-compliance-checker@latest \
  --runtime codex
```

## First run (copy/paste prompt)

```text
Use Campaign QA Supervisor to validate this campaign before launch:
- Campaign ID: spring-launch-2026
- Channel: Paid Social
- Required tracking fields: utm_source, utm_medium, utm_campaign

Return:
1) Overall status (pass/warn/fail)
2) Critical failures
3) Warnings
4) Required remediation actions
5) A short alert message if status is fail
```

## What good output looks like

- Every failure includes evidence and reason.
- Critical blockers are clearly separated from warnings.
- Action list is specific and fix-ready.
- Status is explicit: pass, warn, or fail.

## Beginner safety checklist

- Do not launch on `fail`.
- Require human approval for any override.
- Keep policy checks active even if deadlines are tight.

## Production preflight command

```bash
./bin/skills-hub run-agent \
  --agent marketing/campaign-qa-supervisor \
  --bindings agents/marketing/campaign-qa-supervisor/config/\
tool-bindings.example.json \
  --memory agents/marketing/campaign-qa-supervisor/config/\
memory-profile.example.json \
  --governance agents/marketing/campaign-qa-supervisor/config/\
governance.example.json \
  --approve-live \
  --audit-log ./tmp/campaign-qa-supervisor-run.json
```
