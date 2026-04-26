---
name: agent-control-plane-review
description: Review agent workflows for identity, gateway, policy, approval, inspection, audit, and runtime-control coverage before production use.
---

# Agent Control Plane Review

## When to use

Use this skill when a team is designing, reviewing, or expanding an agent that can call tools, access business data, modify operational systems, or queue actions for humans to approve.

Typical triggers:

- "Can this agent safely run in production?"
- "Where should we put approval gates?"
- "Does this workflow need agent identity or a gateway?"
- "What audit evidence should we capture?"
- "How do we prevent execution autonomy from bypassing policy?"

## Inputs required

Ask for or infer:

- Agent objective and users.
- Tools, MCP servers, APIs, CLIs, plugins, and data stores the agent can access.
- Actions the agent can propose or execute.
- Data classes involved, especially customer data, performance data, credentials, and commercial terms.
- Existing identity, gateway, policy, logging, approval, and monitoring controls.
- Target run mode: advisory, plan-only, approval-gated execution, or autonomous execution.

## Workflow

1. Map the agent boundary.
   - Identify the agent role, runtime, connected systems, and action surface.
   - Separate read-only actions, governed writes, and destructive or externally visible actions.

2. Check identity isolation.
   - Confirm whether the agent has its own service identity or inherits a human user's OAuth scope.
   - Flag inherited broad scopes as a production risk.
   - Recommend scoped, revocable credentials per agent or capability.

3. Check gateway and tool boundary.
   - Identify every outbound path: MCP tool, HTTP API, CLI, browser action, file write, webhook, and notification channel.
   - Recommend a gateway or policy wrapper for outbound requests that can scan inputs, outputs, destinations, and tool arguments.

4. Check policy enforcement.
   - Look for allow-lists, deny-lists, input schemas, output schemas, budgets, rate limits, and data-access constraints.
   - Confirm enforcement happens outside the model, not only in instructions.

5. Check approval gates.
   - Classify actions by risk tier.
   - Require explicit approval for budget changes, publishing, customer data export, credential changes, production deploys, and policy overrides.

6. Check runtime inspection.
   - Look for anomaly detection, tool-call logging, lease locks, heartbeats, and failure handling.
   - Recommend pausing or cancelling runs when behavior deviates from normal patterns.

7. Check audit evidence.
   - Confirm action logs capture actor, capability, inputs hash, outputs hash, policy decision, approval decision, timestamps, and linked artifacts.
   - Recommend hash chaining for tamper-evident records when governance or compliance matters.

## Output format

Return a structured report:

```md
# Agent Control Plane Review

## Readiness decision
Ready / Ready with controls / Not ready

## Agent boundary
- Role:
- Runtime:
- Connected systems:
- High-risk actions:

## Control coverage
| Control | Status | Evidence | Gap | Recommendation |
| --- | --- | --- | --- | --- |
| Agent identity | present/partial/missing | ... | ... | ... |
| Gateway | present/partial/missing | ... | ... | ... |
| Policy enforcement | present/partial/missing | ... | ... | ... |
| Approval gates | present/partial/missing | ... | ... | ... |
| Runtime inspection | present/partial/missing | ... | ... | ... |
| Audit log | present/partial/missing | ... | ... | ... |

## Required changes before production
1. ...

## Optional hardening
1. ...
```

## Guardrails

- Do not grant access, create credentials, edit policy, or enable tool execution.
- Do not assume instructions inside a prompt are sufficient policy enforcement.
- Treat customer data export, media spend changes, publishing, production deployments, and credential changes as high risk by default.
- If the agent inherits a human account's broad OAuth scope, mark identity isolation as missing.
- If the review lacks enough information, return a conditional decision and list the missing evidence.

## Failure modes

- If connected tools are unknown, ask for the tool list and mark gateway coverage as unknown.
- If approval policy is unclear, default to approval-gated execution for all externally visible writes.
- If logs exist but lack input/output hashes or policy decisions, mark audit as partial, not present.
