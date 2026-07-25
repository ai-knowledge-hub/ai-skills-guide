# Getting Started

## Track A: Marketing Practitioner (no coding required)

1. Choose one entry matching your workflow:
   - `skills/` for task-level expertise
   - `agents/` for orchestrated templates
   - `plugins/` for installable bundles that resolve dependencies
   - `tools-mcp/` for integration connectors
2. Check the entry's usability label. Prefer `usable-now`, or follow every item under `requires_setup`.
3. Copy or install the module spec (`SKILL.md`, `AGENT.md`, `plugin.json`, or `TOOL.md`) into your runtime workflow.
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
