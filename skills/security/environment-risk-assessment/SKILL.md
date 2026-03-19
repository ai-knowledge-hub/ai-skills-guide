---
name: environment-risk-assessment
description: Evaluate whether an agent runtime is isolated enough for the
  current task and flag unsafe mixes of credentials, mounts, and privileges.
---

# Environment Risk Assessment

## When to use
- Use before enabling autonomous or semi-autonomous execution.
- Use when an agent runs on a developer workstation.
- Use when production credentials or broad mounts may be present.

## Inputs required
- runtime description
- mounted paths
- credential sources
- network and approval settings

## Workflow
1. Identify whether the runtime is isolated or shared.
2. Review credentials, mounts, and network scope.
3. Flag unsafe mixes such as prod credentials on a laptop or broad writable
   mounts.
4. Recommend safer boundaries and required approvals.

## Output format
- Environment Summary
- High-Risk Conditions
- Recommended Isolation Changes
- Approval Recommendation

## Guardrails
- Default to conservative risk ratings when evidence is incomplete.
- Prefer recommend-and-escalate over automated reconfiguration.
- Cite the observed environment signals for each risk call.
- Do not claim an environment is safe without explicit evidence.

## Failure modes
- If the runtime description is partial, mark confidence as low.
- If protected paths are writable, require human review.
- If production credentials are detected, escalate immediately.
