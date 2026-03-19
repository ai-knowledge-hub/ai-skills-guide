---
name: harness-skill-proposal
description: Turn recurring harness failures into candidate skill or policy
  updates while keeping proposals limited to harness-owned artifacts.
---

# Harness Skill Proposal

## When to use
- Use after run reflection identifies repeated harness failures.
- Use when a new harness rule or skill is needed.
- Use before creating a harness-only PR.

## Inputs required
- reflection findings
- current harness policies
- current skills affected by the failure pattern

## Workflow
1. Read the reflection evidence and isolate a recurring pattern.
2. Decide whether the fix belongs in a skill, a policy file, or a benchmark.
3. Draft the proposed change in a structured, reviewable format.
4. Confirm the target stays within harness-owned paths.

## Output format
- Problem Statement
- Proposed Harness Change
- Target Artifacts
- Expected Improvement
- Approval Notes

## Guardrails
- Only target harness-owned artifacts by default.
- Do not propose edits to production code or deployment config.
- Tie every proposal to concrete run evidence.
- Escalate if the best fix would require protected-path changes.

## Failure modes
- If evidence is too thin, recommend more benchmarks instead of a change.
- If multiple fixes are plausible, present the smallest safe proposal first.
- If scope expands beyond harness paths, stop and require approval.
