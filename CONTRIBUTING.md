# Contributing

## Who this is for

- Marketing practitioners creating reusable workflows
- Ad-tech engineers hardening workflows with scripts/tools
- Software engineers building maintenance and QA automations
- Security teams defining safe agent behaviors and review packs
- Agent builders maintaining evolving harnesses and guardrails

## Pull request requirements

1. Add/update one entry under one module:
   - `skills/<domain>/<slug>`
   - `agents/<domain>/<slug>`
   - `tools-mcp/<domain>/<slug>`
2. Include spec file (`SKILL.md` or `AGENT.md` or `TOOL.md`),
   `README.md`, `tests/test-prompts.md`, and `examples/`.
3. Include correct manifest (`skill.yaml`, `agent.yaml`, or `tool.yaml`).
4. Ensure at least 5 realistic prompts with expected behavior.
5. Document assumptions (APIs, data sources, required tools).
6. Include risk notes if shell commands, writes, or publishing actions are
   involved.
7. For security or agentops entries, document approval boundaries and
   paths that remain read-only by default.

## Module folder standard

```text
<entry-slug>/
  README.md
  SKILL.md | AGENT.md | TOOL.md
  skill.yaml | agent.yaml | tool.yaml
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
- Risky skills default to analyze, halt, or escalate before write actions.
- Agentops entries only target harness-owned artifacts unless a human
  explicitly approves broader scope.

## Quality gates

Run before opening a PR:

```bash
bash scripts/validate-skills.sh
go run ./cmd/registry-builder
bash scripts/validate-manifests.sh
```

If your changes touch `apps/web`, also run:

```bash
cd apps/web
pnpm test:e2e
```

If Playwright browsers are not installed yet, run `pnpm test:e2e:setup`
first (see `apps/web/README.md`).

## Branch and commit guidance

- Default flow in this repo is `dev` -> `main`.
- Keep commits on `dev` focused and reviewable; merge into `main` via PR.
- Use scoped commit messages (e.g., `feat(skill): add pmax creative workshop`).
