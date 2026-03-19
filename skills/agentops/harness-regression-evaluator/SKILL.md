---
name: harness-regression-evaluator
description: Test proposed harness updates against saved prompts and
  red-team cases to reject regressions before adoption.
---

# Harness Regression Evaluator

## When to use
- Use before accepting a harness skill or policy change.
- Use when a proposed update claims to improve safety or reliability.
- Use to compare baseline and candidate harness behavior.

## Inputs required
- proposed harness change summary
- benchmark prompts
- red-team prompts
- pass/fail criteria

## Workflow
1. Load the baseline expectations and proposed change.
2. Run or simulate the benchmark and red-team prompts.
3. Compare outcomes against pass/fail criteria.
4. Approve, reject, or require more evidence.

## Output format
- Evaluation Summary
- Passed Scenarios
- Failed Scenarios
- Regression Risks
- Recommendation

## Guardrails
- Reject changes that weaken safety or widen scope silently.
- Keep acceptance criteria explicit and deterministic.
- Separate benchmark wins from red-team regressions.
- Require human review if the evaluator cannot reproduce results.

## Failure modes
- If benchmark inputs are missing, do not approve the change.
- If results are mixed, prefer rejection or more evidence.
- If the proposal touches protected paths, require manual review.
