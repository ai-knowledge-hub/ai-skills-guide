---
name: ad-platform-policy-gate-designer
description: Design approval gates, auto-execution thresholds, deny rules, and audit requirements for DV360, Google Ads, and ad-platform agent writes.
---

# Ad Platform Policy Gate Designer

## When to use

Use this skill when a team needs concrete governance rules for an agent that can propose or execute ad-platform changes.

Typical triggers:

- "What Google Ads actions can this agent auto-execute?"
- "Which DV360 changes need human approval?"
- "Design policy gates for bid, budget, and targeting changes."
- "Create deny rules for regulated categories or sensitive markets."
- "Turn our campaign-risk policy into executable thresholds."

## Inputs required

Collect:

- Platforms and entity types: account, advertiser, campaign, insertion order, line item, ad group, keyword, audience, feed item.
- Action types: bid, budget, status, targeting, audience, creative, feed, tracking.
- Business context: client, market, spend tier, regulated category, SLA.
- Existing approval owners and escalation paths.
- Risk thresholds: budget delta, bid delta, audience expansion, geo expansion, pacing variance.
- Rollback and audit requirements.

## Workflow

1. Classify actions by risk.
   - Low: read-only, recommendation-only, status summaries, small reversible bid changes.
   - Medium: bounded bid/budget edits, narrow targeting edits, non-publishing feed drafts.
   - High: budget increases, targeting broadening, audience expansion, live publishing, tracking changes.
   - Blocked: cross-client actions, unscoped exports, forbidden categories, policy overrides by agent.

2. Convert risk into gates.
   - Auto-execute only when action is low risk, scoped, reversible, and below thresholds.
   - Route medium/high risk to named approvers.
   - Deny actions that violate scope or organisational policy.

3. Define bounded execution grants.
   - Include workspace, platform, advertiser, entity, allowed action, max delta, expiry, and approval id.

4. Define audit evidence.
   - Require current state, proposed diff, policy decision, approval decision, execution payload, post-write state, and rollback patch.

5. Produce implementation-ready policy tables.

## Output format

```md
# Ad Platform Policy Gates

## Risk model
| Risk tier | Definition | Examples |
| --- | --- | --- |

## Policy gates
| Action | Auto-execute | Approval required | Deny when | Audit evidence |
| --- | --- | --- | --- | --- |

## Thresholds
| Metric | Low-risk range | Approval threshold | Block threshold |
| --- | --- | --- | --- |

## Bounded grant fields
- workspace_id
- platform
- advertiser_id
- entity_type
- entity_id
- allowed_action
- max_delta
- expires_at
- approval_id

## Engineering notes
1. ...
```

## Guardrails

- Do not approve or execute live campaign changes.
- Do not make policy less strict than the user's stated organisational baseline.
- Treat missing client/workspace/advertiser scoping as a blocker for auto-execution.
- Require approval for targeting expansion, budget increases, tracking changes, and regulated-category changes by default.
- Recommend denial when rollback or post-write verification is unavailable for risky writes.
