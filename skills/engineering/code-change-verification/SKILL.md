---
name: code-change-verification
description: Select and run the right verification commands after code
  changes, then summarize failures, flakiness, and remaining risk.
---

# Code Change Verification

## When to use
- Use after editing runtime code, tests, examples, or build files.
- Use when a reviewer asks for the exact verification status of a change.
- Use when a task needs stack-aware command selection.

## Inputs required
- changed files or diff summary
- detected stack or package manager
- available local verification commands
- any known flaky tests or skipped checks

## Workflow
1. Map changed files to the relevant subsystem and stack.
2. Select lint, typecheck, test, and build commands using the command matrix.
3. Run the smallest sufficient command set first, then expand if failures
   suggest broader impact.
4. Record pass, fail, skip, or flaky for each command.
5. Summarize what passed, what failed, and what still needs human review.

## Output format
- Changed Scope
- Commands Run
- Results
- Failures and Likely Causes
- Remaining Risk

## Guardrails
- Do not claim a change is verified if commands were skipped.
- Distinguish missing tooling from passing verification.
- Keep verification read-only except for sandboxed local command execution.
- Escalate if the required command set is unavailable.

## Failure modes
- If tooling is missing, report the exact missing binary or script.
- If tests are flaky, mark them as flaky instead of pass.
- If only partial verification ran, say so explicitly.
