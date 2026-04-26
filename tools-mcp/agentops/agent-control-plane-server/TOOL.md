# Agent Control Plane MCP Server

## Purpose

This is a template for an MCP server that keeps agent execution governed. It does not run marketing actions itself. It checks whether proposed actions are allowed, records what happened, and routes risky actions through approval.

## Capabilities

### `check_policy`

Checks a proposed action against a policy document.

Input shape:

```json
{
  "agent_id": "campaign-agent",
  "capability": "apply_budget_shift",
  "scope": {"account_id": "act_123", "channel": "google_ads"},
  "risk_tier": "high",
  "inputs_hash": "sha256:..."
}
```

Output shape:

```json
{
  "decision": "allow_with_approval",
  "matched_rules": ["budget-write-requires-approval"],
  "approval_required": true,
  "reason": "Budget writes are high-risk actions."
}
```

### `record_agent_action`

Writes an auditable event for proposed, approved, executed, failed, or rejected actions.

### `request_approval`

Creates a human approval request for high-risk actions and returns an approval ticket identifier.

### `verify_audit_chain`

Checks whether audit events remain ordered and tamper-evident by validating previous-event hashes.

## Runtime assumptions

- Policy lives outside the model.
- The agent calls this server before executing governed actions.
- Action logs are append-only.
- Approval decisions are recorded as separate events.

## Guardrails

- Do not expose raw production APIs through this server.
- Do not let the model override policy decisions.
- Do not allow policy edits through the same agent workflow being governed.
- Require explicit approval for high-risk writes, customer-data exports, publishing, budget changes, and permission changes.
- Log denied actions as well as approved actions.
