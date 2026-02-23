# How to Test Skills

## Test types

1. Activation: skill triggers only on intended requests.
2. Workflow: tool/script sequence is followed.
3. Failure mode: missing data/tool errors handled safely.
4. Output shape: required sections/schema preserved.
5. Guardrails: no fabrication, risky actions require confirmation.
6. UI Smoke (when UI changes): key routes render and critical interactions still work.

## Test evidence format

For each prompt:
- prompt text
- expected behavior
- observed behavior
- pass/fail
- notes

## UI smoke baseline (hub website)

Run from `apps/web`:

```bash
pnpm test:e2e
```

Current smoke scope:
- `/`
- `/skills`
- `/skills/<sample>`
- filter interaction
- copy-button presence/click
