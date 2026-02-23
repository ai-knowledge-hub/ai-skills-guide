# Contributing

## Who this is for

- Marketing practitioners creating reusable workflows
- Ad-tech engineers hardening workflows with scripts/tools

## Pull request requirements

1. Add/update one skill under `skills/marketing` or `skills/adtech`.
2. Include `SKILL.md`, `README.md`, `tests/test-prompts.md`, and `examples/`.
3. Add at least 5 realistic prompts with expected behavior.
4. Document assumptions (APIs, data sources, required tools).
5. Include risk notes if shell commands, writes, or publishing actions are involved.

## Skill folder standard

```text
<skill-name>/
  README.md
  SKILL.md
  scripts/        # deterministic logic
  references/     # optional deep docs
  config/         # optional rules
  examples/       # input/output examples
  tests/
    test-prompts.md
```

## Review checklist

- Routing description is explicit and discoverable.
- Guardrails prevent fabrication and unsafe actions.
- Deterministic logic is scripted, not only prompt-based.
- Failures and fallback paths are documented.
- Output shape is consistent and testable.

## Quality gates

Run before opening a PR:

```bash
bash scripts/validate-skills.sh
```

If your changes touch `apps/web`, also run:

```bash
cd apps/web
pnpm test:e2e
```

If Playwright browsers are not installed yet, run `pnpm test:e2e:setup` first (see `apps/web/README.md`).

## Branch and commit guidance

- Use short feature branches prefixed with `codex/` (e.g., `codex/skill-utm-linter`).
- Use scoped commit messages (e.g., `feat(skill): add pmax creative workshop`).
