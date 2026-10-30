# Getting Started

## Track A: Marketing Practitioner (no coding required)

1. Choose one entry matching your workflow:
   - `skills/` for task-level expertise
   - `agents/` for orchestrated templates
   - `plugins/` for composition templates today and implemented bundles later
   - `tools-mcp/` for integration connectors
2. Check the entry's usability label and limitations. Treat `not-verified` as implemented but unproven for your target; use `setup-required` only after completing every listed setup item.
3. Install instruction or implemented entries only when their availability permits it. For `template-only`, inspect or copy repository source as a scaffold without runtime registration.
4. Run the prompts in `tests/test-prompts.md`.
5. Evaluate output consistency against expected format.
6. Tune wording and constraints, then re-test.

## Track B: Ad-Tech Engineer

1. Start with the same steps as Track A.
2. Move deterministic logic into `scripts/`.
3. Add strict input validation and failure handling.
4. Add sample data files under `examples/`.
5. Open a PR with test evidence.

## Runtime notes

These skills are authored for Agent Skills-style runtimes and can be adapted to Codex, Claude-style, and similar ecosystems.

## Hub UI quick check

If you are working on the website catalog:

1. `cd apps/web`
2. `pnpm install`
3. `pnpm dev`
4. `pnpm test:e2e`

Primary routes:
- `/skills`
- `/agents`
- `/tools-mcp`
- `/plugins`

Read [using-the-catalog.md](using-the-catalog.md) before assuming installation means an external integration is connected.
