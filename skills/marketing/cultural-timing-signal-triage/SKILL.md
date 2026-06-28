---
name: cultural-timing-signal-triage
description: Triage live cultural moments for brand fit, timing, risk, usefulness, and approval path before a response is created.
---

# Cultural Timing Signal Triage

## When to use
Use this skill when a team wants to respond to a live cultural moment, fandom, meme, crisis, creator trend, sports event, or market conversation.

## Inputs required
- Description of the cultural signal
- Source and evidence of momentum
- Brand relevance
- Audience overlap
- Proposed response idea
- Timing window
- Risk categories and approval owners

## Workflow
1. Summarize the cultural signal and verify the moment is real enough to consider.
2. Assess brand permission: why this brand has a right to participate.
3. Score timing urgency, audience relevance, usefulness, risk, and execution complexity.
4. Recommend `go`, `watch`, `adapt`, or `do_not_act`.
5. Define response shape, approval route, and expiry time.

## Output format
Return:
- `signal_summary`
- `brand_permission`
- `score_table`
- `recommendation`
- `response_window`
- `approval_route`
- `risk_notes`
- `draft_next_step`

## Guardrails
- Do not recommend opportunistic responses to tragedy, harm, or sensitive crises without explicit ethical review.
- Prefer useful contribution over brand insertion.
- Flag low brand permission even if the moment is popular.
- Include an expiry time; live moments decay.
