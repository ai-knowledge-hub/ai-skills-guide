---
name: coverage-gap-reporter
description: Interpret coverage reports for changed code and turn them into
  actionable, file-level test recommendations.
---

# Coverage Gap Reporter

## When to use
- Use when a PR or branch has fresh coverage output.
- Use when you need coverage evidence for changed files.
- Use when coverage dropped and the next test target is unclear.

## Inputs required
- coverage report or JSON artifact
- changed files or target module list
- coverage adapter or report format hint

## Workflow
1. Parse the coverage source using the configured adapter.
2. Isolate changed files or requested modules.
3. Report per-file coverage and notable uncovered areas.
4. Translate uncovered areas into concrete test suggestions.
5. Call out missing or incompatible coverage data explicitly.

## Output format
- Coverage Summary
- Changed File Coverage
- Uncovered Risk Areas
- Suggested Tests
- Data Quality Notes

## Guardrails
- Do not guess line coverage when the report is incomplete.
- Distinguish missing coverage data from low coverage.
- Tie recommendations to reported uncovered areas.
- Keep outputs deterministic and file-scoped.

## Failure modes
- If no adapter matches the report, ask for the supported format.
- If changed files are absent from the report, mark the result incomplete.
- If coverage is stale, say so explicitly.
