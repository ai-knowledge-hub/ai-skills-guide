---
name: pr-review-and-draft
description: Review a code diff from multiple perspectives and produce a
  concise PR draft with risks, validation, and follow-up notes.
---

# PR Review and Draft

## When to use
- Use when a branch or diff is ready for review.
- Use when you need a deterministic PR summary.
- Use when a reviewer wants a focused risk assessment.

## Inputs required
- diff or changed file list
- verification results if available
- test or coverage context

## Workflow
1. Review the diff using the rubric for correctness, performance, security,
   and tests.
2. Capture concrete findings with evidence.
3. Summarize validation already completed and what remains.
4. Draft the PR summary with intent, impact, and risk.

## Output format
- Findings
- Open Questions
- Verification Status
- PR Summary
- Risk Notes

## Guardrails
- Prioritize bugs and regressions over style comments.
- Do not claim security or performance review without evidence.
- Keep PR summaries grounded in the actual diff.
- Escalate if the diff affects protected paths or production policy.

## Failure modes
- If the diff is incomplete, say which context is missing.
- If no verification is available, mark review confidence as reduced.
- If the change is too broad, recommend splitting the PR.
