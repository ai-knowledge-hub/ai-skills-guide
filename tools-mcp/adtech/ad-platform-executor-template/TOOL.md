# Ad Platform Executor Template

## Purpose

This template defines a narrow executor for DV360, Google Ads, and similar ad-platform writes. The executor applies only structured, policy-approved changes. It should not expose broad platform APIs directly to an LLM.

## Capabilities

### `read_campaign_state`

Reads current state for a scoped account, advertiser, campaign, line item, ad group, keyword, audience, or feed item.

### `validate_change_plan`

Checks a structured change plan against current platform state, policy decision, approval record, and bounded execution grant.

### `apply_approved_change`

Applies one approved bid, budget, targeting, status, feed, or tracking change through the relevant platform API.

### `apply_bulk_targeting_change`

Applies a validated bulk targeting diff. For DV360, queue updates to the same line item and prefer bulk edit operations where supported.

### `rollback_last_execution`

Uses stored pre-image and rollback patch to revert the last execution when supported by the platform and policy.

## Expected input shape

```json
{
  "change_plan_id": "plan_123",
  "execution_grant": {
    "workspace_id": "client-123",
    "platform": "dv360",
    "advertiser_id": "987654",
    "entity_type": "line_item",
    "entity_id": "li_456",
    "allowed_action": "apply_bid_delta",
    "max_delta_percent": 10,
    "expires_at": "2026-05-06T18:00:00Z",
    "approval_id": "approval_abc"
  },
  "proposed_diff": {
    "field": "bid",
    "from": 1.2,
    "to": 1.08,
    "delta_percent": -10
  }
}
```

## Required execution sequence

1. Resolve the scoped execution grant.
2. Retrieve current platform state.
3. Confirm the proposed diff still applies to current state.
4. Confirm policy and approval are valid.
5. Apply the API write.
6. Verify post-write state.
7. Store audit event and rollback patch.

## Guardrails

- Do not accept free-form natural-language write requests.
- Do not expose raw Google refresh tokens, service-account keys, or API credentials to the agent runtime.
- Do not execute without a valid scoped grant and policy decision.
- Do not execute medium/high-risk changes without approval evidence.
- Do not run concurrent writes against the same DV360 line item; queue or merge them.
- Do not hide API failures. Return structured failure reasons and preserve pre-image state.
