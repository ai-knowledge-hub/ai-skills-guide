# Getting Started

## Track A: Marketing Practitioner (no coding required)

1. Choose one skill matching your repeated workflow.
2. Copy `SKILL.md` into your runtime skill directory.
3. Run the prompts in `tests/test-prompts.md`.
4. Evaluate output consistency against expected format.
5. Tune wording and constraints, then re-test.

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
