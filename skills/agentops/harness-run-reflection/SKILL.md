---
name: harness-run-reflection
description: Inspect run logs and task outcomes to identify recurring
  harness failures, skipped skills, and inefficient control-flow patterns.
---

# Harness Run Reflection

## When to use
- Use after a batch of agent runs or benchmark tasks.
- Use when skills are being skipped or invoked too late.
- Use before proposing harness policy changes.

## Inputs required
- recent run logs
- task outcomes or benchmark summaries
- current skill and policy references

## Workflow
1. Review the run logs for repeated failure patterns.
2. Identify missed skills, inefficient loops, and approval bypass attempts.
3. Group the findings by frequency and severity.
4. Recommend harness improvements without editing production code.

## Output format
- Run Summary
- Recurring Failure Patterns
- Skipped or Late Skills
- Candidate Harness Improvements
- Scope Notes

## Guardrails
- Limit analysis to harness behavior and policy usage.
- Do not recommend edits to production code as harness fixes.
- Cite specific run-log evidence for every pattern.
- Escalate if logs suggest policy violations or exfiltration attempts.

## Failure modes
- If logs are incomplete, lower confidence explicitly.
- If the root cause spans production code and harness logic, split the
  recommendation into separate tracks.
- If the proposed fix touches protected paths, require human review.
