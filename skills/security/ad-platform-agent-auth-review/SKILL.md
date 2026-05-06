---
name: ad-platform-agent-auth-review
description: Review DV360, Google Ads, and similar marketing agents for unsafe OAuth, broad platform access, weak token handling, missing read/write separation, and missing approval gates.
---

# Ad Platform Agent Auth Review

## When to use

Use this skill when a team is building or reviewing an agent that connects to DV360, Google Ads, Meta Ads, Merchant Center, product feeds, analytics warehouses, or other marketing platforms with OAuth, API keys, or service credentials.

Typical triggers:

- "Our agent uses a user's Google OAuth token; is that safe?"
- "Can this DV360 agent move from read-only to write access?"
- "How should we scope Google Ads access for an optimisation agent?"
- "What auth risks exist before this agent changes bids, budgets, or targeting?"
- "Should this agent use personal OAuth, service account, or brokered access?"

## Inputs required

Collect:

- Platforms connected: DV360, Google Ads, Meta Ads, Merchant Center, warehouse, CRM, CMS.
- Credential type: personal OAuth, service account, API key, app token, internal execution grant.
- Requested scopes and actual platform permissions.
- Advertiser/workspace/client mapping.
- Actions: read, propose, approve, execute, rollback.
- Token storage location and rotation process.
- Whether the LLM can see tokens, tool outputs, or raw API credentials.
- Current approval and audit flow.

## Workflow

1. Map the credential chain.
   - Identify who grants access, where refresh credentials live, and which component retrieves execution credentials.
   - Flag any path where the LLM sees long-lived credentials.

2. Check scope and workspace isolation.
   - Compare token/platform access with the agent's actual task.
   - Flag access that crosses client, brand, market, advertiser, or workspace boundaries without explicit mapping.

3. Check identity model.
   - Identify whether API writes appear as a human user, service identity, or agent-specific identity.
   - Flag misleading audit trails where a human identity masks an agent action.

4. Check read/write separation.
   - Confirm read-only analysis and write-capable execution use separate permissions and components.
   - Flag single-token designs that can both inspect and mutate campaigns.

5. Check approval readiness.
   - Identify high-risk actions: budget increases, bid changes, targeting expansion, audience edits, product-feed updates, tracking changes, publishing.
   - Require step-up approval or bounded execution grants for these actions.

6. Check audit and rollback readiness.
   - Confirm the system records current state, proposed diff, policy decision, approver, execution payload, post-write verification, and rollback patch.

## Output format

```md
# Ad Platform Agent Auth Review

## Decision
Read-only safe / Plan-only only / Approval-gated writes possible / Blocked

## Auth risk summary
- Credential model:
- Scope fit:
- Workspace isolation:
- Token handling:
- Audit identity:

## Findings
| Area | Status | Evidence | Required change |
| --- | --- | --- | --- |

## Recommended trust zone design
| Zone | Component | Allowed access |
| --- | --- | --- |
| User identity | ... | ... |
| Control plane | ... | ... |
| Agent runtime | ... | ... |
| Execution zone | ... | ... |

## Required before write access
1. ...
```

## Guardrails

- Do not approve OAuth scopes or platform access.
- Do not create, rotate, expose, or request secrets.
- Do not recommend direct LLM access to long-lived refresh tokens, service keys, or API credentials.
- Treat personal OAuth used for agent writes as a production risk unless mediated by a broker and scoped per workspace.
- Treat bid, budget, targeting, audience, product-feed, and tracking changes as write-risk actions by default.

## Failure modes

- If scopes are unknown, mark scope fit as unknown and recommend plan-only/read-only mode.
- If platform permissions exceed the agent task, recommend broker-level scoping before writes.
- If audit identity records only a human user for agent actions, mark audit identity as partial or misleading.
