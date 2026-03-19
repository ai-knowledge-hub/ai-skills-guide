---
name: test-gap-analyzer
description: Review changed code and current tests to identify missing
  scenarios, weak assertions, and regression blind spots.
---

# Test Gap Analyzer

## When to use
- Use after a code change when you want to know what tests are still missing.
- Use during PR review to assess test completeness.
- Use when failures hint at an untested edge case.

## Inputs required
- changed files or diff summary
- current tests covering those files
- known product edge cases or failure reports

## Workflow
1. Inspect changed code paths and existing tests.
2. Map missing coverage by scenario type: unit, integration, regression,
   negative path, and edge case.
3. Assess assertion quality and setup realism.
4. Prioritize missing tests by defect risk.
5. Suggest next tests in a deterministic structure.

## Output format
- Coverage Summary
- Missing Scenarios
- Weak Assertions
- Highest-Risk Gaps
- Recommended Next Tests

## Guardrails
- Do not invent coverage data that was not observed.
- Separate code-risk analysis from measured test coverage.
- Tie each suggested scenario to specific code behavior.
- Escalate if no tests are available to inspect.

## Failure modes
- If tests are absent, label the change as high-risk.
- If fixtures are incomplete, call out the exact missing setup.
- If code behavior is ambiguous, mark confidence as low.
