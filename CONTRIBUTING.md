# Contributing

## Scope

Contribute reusable skills for marketing practitioners and ad-tech engineers.

## Pull Request Requirements

1. Add or update one skill folder under `skills/`.
2. Include a `SKILL.md` with clear routing description and guardrails.
3. Include `tests/test-prompts.md` with at least 3 realistic prompts.
4. Document required tools, APIs, and runtime assumptions.
5. Add risk notes for security-sensitive actions (shell, data writes, external publishing).

## Skill Folder Standard

```text
<skill-name>/
  SKILL.md
  scripts/        # optional deterministic logic
  references/     # optional long-form guidance
  config/         # optional policy/config files
  tests/
    test-prompts.md
```

## Review Checklist

- Does the description reliably trigger for the intended intent?
- Are deterministic calculations moved to scripts where needed?
- Are failure modes and non-fabrication rules explicit?
- Are tool permissions and risky actions clearly constrained?
- Are outputs structured and testable?
