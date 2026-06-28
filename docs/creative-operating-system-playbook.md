# Creative Operating System Playbook

This playbook shows how to use the Creative Operating System pack as a practical build sequence.

## The operating model

A creative operating system has seven working parts:

1. **Memory**: approved brand knowledge, prior campaign learnings, claims evidence, creator learnings, and examples.
2. **Context**: audience, category, culture, media behavior, product reality, and constraints.
3. **Utility**: the human problem the work helps with.
4. **Product surfaces**: owned places where the brand can do something useful, not only say something.
5. **Creator collaboration**: human distribution, trust, format fluency, and cultural interpretation.
6. **Evaluation**: repeatable scoring for quality, usefulness, distinctiveness, evidence, and risk.
7. **Governance**: approvals, claims checks, timing checks, and escalation paths.

## Build sequence

### 1. Build or refresh memory

Use `adtech/brand-rag-memory-bootstrap` to collect the minimum useful memory base:

- brand strategy
- tone of voice
- past winning and failed campaigns
- approved claims
- banned claims
- audience research
- creator learnings
- post-campaign reviews

### 2. Audit the system

Run `marketing/creative-operating-system-audit`.

Expected output:

- maturity score
- evidence gaps
- governance gaps
- 30-day action plan

### 3. Design around utility

Run `marketing/utility-campaign-concept-designer`.

Ask for concepts that help people do something specific. If the idea cannot name the human friction, it is probably only content.

### 4. Map owned surfaces

Run `marketing/product-as-media-mapper`.

Look for places where the brand can deliver the idea through product, service, app, commerce, CRM, retail, support, packaging, or data.

### 5. Triage timing

Run `marketing/cultural-timing-signal-triage` for any live moment or trend.

Require an explicit decision:

- act now
- prepare and wait
- monitor
- decline

### 6. Brief creators

Run `marketing/creator-strategy-brief`.

The brief should define the creator's strategic role, not just ask them to distribute brand copy.

### 7. Score the work

Run `marketing/ai-output-eval-scorecard` with these optional dimensions enabled:

- utility
- product-as-media potential
- cultural timing
- human meaning
- brand memory
- distinctiveness
- creator fit

## Minimum launch gate

Before anything public ships, confirm:

- claims are evidenced
- sensitive context is reviewed
- creator disclosures are clear
- product changes are approved
- measurement is defined
- rollback or pause path exists

If those gates are missing, the output is not launch-ready even if the copy reads well.
