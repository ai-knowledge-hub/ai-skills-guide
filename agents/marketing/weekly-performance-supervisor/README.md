# Weekly Performance Supervisor

Agent template for deterministic weekly performance reporting with
QA-gated publish control.

## What this agent does

- Orchestrates weekly reporting in a fixed sequence.
- Builds dashboard sections from normalized data.
- Runs QA checks before publish.
- Generates executive narrative only when QA passes.

## Operational metadata

- Role: weekly reporting supervisor for QA-gated performance review workflows
- Autonomy level: semi-autonomous
- Approval boundary: may coordinate reporting workflows and draft narratives after QA approval; require human approval before any external publish or stakeholder send
- Outputs:
  - publish decision
  - dashboard package
  - QA package
  - executive narrative

## Before you start

1. Install this agent package.
2. The installer resolves and pins the required skills and provider packages.
3. Follow `.runtime/<runtime>.json` to configure GA4 and BigQuery.
4. Confirm date windows and required channels. Bundled memory and governance
   profiles require no manual binding edits.

Install commands (Codex example):

```bash
./bin/skills-hub install \
  --module agents \
  --entry marketing/weekly-performance-supervisor@latest \
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
  --runtime codex \
  --model-attestation "$MODEL_ATTESTATION_PATH" \
  --approve-live \
  --audit-log ./tmp/weekly-performance-supervisor-run.json
```
