---
name: advertising-signal-ownership-auditor
description: Audit who collects, controls, receives, retains, and uses advertising and commerce signals across pixels, server events, campaign platforms, CRM, orders, returns, and support systems. Use when designing measurement ownership, first-party evidence, platform optimization, or agentic advertising governance.
---

# Advertising Signal Ownership Auditor

## When to use
Use this skill for the situations described in its frontmatter routing description.

## Workflow
1. Inventory every event from exposure through post-purchase outcome.
2. Identify the collector, controller, processor, visible parties, and system of record.
3. Record identifiers, consent basis, retention, and allowed uses.
4. Mark whether each signal trains platform optimization, supports independent evaluation, or both.
5. Identify duplicate collection, inaccessible evidence, missing post-conversion outcomes, and circular validation.
6. Require a canonical source for business outcomes.
7. Return a ledger conforming to `shared/schemas/advertising/signal-ownership-ledger.schema.json`.

## Read when needed
- Use `references/ownership-questions.md` during stakeholder discovery.

## Guardrails
- Do not treat platform reporting as independent validation.
- Do not assume the advertiser owns or can export a platform-derived signal.
- Do not recommend collecting data without a documented purpose and consent basis.
- Never place secrets or raw personal data in the ledger.
