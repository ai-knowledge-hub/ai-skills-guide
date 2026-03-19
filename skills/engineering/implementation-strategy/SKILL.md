---
name: implementation-strategy
description: Read the repository, map impacted files and commands, then
  write a change strategy before editing code. Use for non-trivial code
  maintenance tasks or any request that may affect core runtime behavior.
---

# Implementation Strategy

## When to use
- Use before editing runtime code, API surfaces, or shared libraries.
- Use when the request spans multiple files or subsystems.
- Use when you need to map commands, risks, and likely side effects.

## Inputs required
- user request or issue summary
- repository root and relevant module path
- available verification commands
- any local policy files such as `AGENTS.md` or `CONTRIBUTING.md`

## Workflow
1. Read repo policy and locate the relevant directories and tests.
2. Identify impacted files, likely dependencies, and blast radius.
3. List the canonical verification commands for the touched stack.
4. Summarize the proposed edit strategy before writing code.
5. Flag approvals needed for risky scope changes.

## Output format
- Task Summary
- Impacted Areas
- Planned Approach
- Verification Commands
- Risks and Approval Gates

## Guardrails
- Do not start editing before the plan is explicit.
- Keep the plan scoped to the request and current evidence.
- Cite specific files or commands when you infer impact.
- Escalate if required commands or repo rules are missing.

## Failure modes
- If repo structure is unclear, ask for the missing context.
- If canonical commands are absent, recommend a safe fallback list.
- If the request implies deploy or infra changes, require human review.
