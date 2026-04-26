---
name: marketing-agent-risk-review
description: Review marketing and adtech agents for spend, publishing, customer-data, audience, attribution, and compliance risks before deployment.
---

# Marketing Agent Risk Review

## When to use

Use this skill when a proposed agent touches marketing operations, commerce, analytics, paid media, lifecycle CRM, SEO, product feeds, or content publishing.

Typical triggers:

- "Can this paid media agent safely change budgets?"
- "What should be approval-gated in this marketing workflow?"
- "Is this analytics agent allowed to export customer data?"
- "What security risks exist in this campaign copilot?"
- "How should we separate suggestion, approval, and execution?"

## Inputs required

Collect:

- Agent role and business objective.
- Channels involved: search, social, programmatic, CRM, commerce, SEO, analytics, CMS.
- Tool list and permissions.
- Data categories: aggregate metrics, audience lists, customer records, creative assets, product catalog, commercial terms.
- Actions the agent may perform.
- Human review points and approval owners.
- Logging and retention requirements.

## Workflow

1. Classify the action surface.
   - Low risk: read aggregate metrics, summarize reports, draft recommendations.
   - Medium risk: generate campaign drafts, create audience suggestions, prepare product-feed changes.
   - High risk: change budgets, publish content, modify bidding, export user-level data, update customer journeys, alter tracking, or change permissions.

2. Classify the data surface.
   - Aggregate campaign metrics are lower risk.
   - Customer-level data, audience membership, CRM events, cookies, identifiers, conversion values, and commercial terms require stricter controls.

3. Check tool permissions.
   - Identify whether each tool is read-only, governed write, or unrestricted write.
   - Flag broad API scopes and human-token inheritance.

4. Check approval boundaries.
   - Require human approval for all high-risk actions.
   - Require a second reviewer for data export, production publishing, bidding changes, or policy overrides.

5. Check compliance and brand risk.
   - Identify claims, regulated categories, targeting constraints, attribution claims, and platform policy exposure.
   - Recommend deterministic checks where rules are clear.

6. Check audit and rollback.
   - Confirm every decision can be traced to prompt, model, tool call, policy result, approver, and artifact version.
   - Recommend rollback plans for published or budget-impacting actions.

## Output format

```md
# Marketing Agent Risk Review

## Overall risk
Low / Medium / High / Blocked

## Risk matrix
| Area | Risk | Evidence | Required control |
| --- | --- | --- | --- |
| Spend | ... | ... | ... |
| Publishing | ... | ... | ... |
| Customer data | ... | ... | ... |
| Tool permissions | ... | ... | ... |
| Compliance | ... | ... | ... |
| Audit and rollback | ... | ... | ... |

## Approval gates
1. ...

## Required changes
1. ...

## Safe starting mode
Advisory / Plan-only / Approval-gated execution / Not recommended
```

## Guardrails

- Do not approve or execute campaign, bidding, publishing, audience, or data-export actions.
- Treat spend changes, customer-level data, and production publishing as high risk by default.
- Do not rely on the same model instance to both generate and approve regulated or brand-sensitive content.
- If tool permissions are unclear, classify the tool as unrestricted until proven otherwise.
- Recommend plan-only mode when identity, policy, logging, or approval boundaries are missing.

## Failure modes

- If platform policies are unknown, require policy review before launch.
- If data classification is missing, block customer-level export and recommend aggregate-only operation.
- If rollback is impossible, require stricter approval and staging before execution.
