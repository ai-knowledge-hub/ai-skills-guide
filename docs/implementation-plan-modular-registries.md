# Modular Registries Implementation Plan

## Objective
Implement one repository with three route modules and three registry indexes:
- `/skills`
- `/agents`
- `/tools-mcp`

## Phases

### Phase 1: Architecture and Contracts
- [x] Create architecture spec for three modules and index outputs.
- [x] Define module naming and directory conventions.
- [x] Define index file names and ownership boundaries.

### Phase 2: Manifest Schemas
- [x] Keep existing `shared/schemas/skill.schema.json`.
- [x] Add `shared/schemas/agent.schema.json`.
- [x] Add `shared/schemas/tool.schema.json`.
- [x] Wire schema validation scripts to include agent and tool manifests.

### Phase 3: Registry Builder Refactor
- [x] Introduce module-aware registry models in `internal/registry/types.go` and builder functions.
- [x] Implement `BuildSkillsIndex`, `BuildAgentsIndex`, `BuildToolsIndex`.
- [x] Emit:
  - `registry/skills-index.json`
  - `registry/agents-index.json`
  - `registry/tools-index.json`
- [x] Add `--module skills|agents|tools|all` to `cmd/registry-builder/main.go`.

### Phase 4: Web Routing and Data Layer
- [x] Refactor `apps/web/lib/registry.ts` into module-aware loaders.
- [x] Add routes:
  - `apps/web/app/agents/page.tsx`
  - `apps/web/app/agents/[...id]/page.tsx`
  - `apps/web/app/tools-mcp/page.tsx`
  - `apps/web/app/tools-mcp/[...id]/page.tsx`
- [x] Update homepage/navigation to expose all three modules.

### Phase 5: Validation and CI
- [ ] Generalize `scripts/validate-skills.sh` to support all modules.
- [x] Extend `scripts/validate-manifests.sh` for agent/tool manifests.
- [ ] Add unit tests for parsers/builders.
- [x] Update web smoke tests for new routes.

### Phase 6: Seed Content
- [x] Add initial `agents/` entries (minimum 3).
- [x] Add initial `tools-mcp/` entries (minimum 3).
- [x] Ensure examples/tests exist for each new entry.

### Phase 7: Deployment and Rollout
- [ ] Route `skills.ai-knowledge-hub.org` to `/skills`.
- [ ] Route `agents.ai-knowledge-hub.org` to `/agents`.
- [ ] Decide whether to keep `/tools-mcp` under skills domain or launch dedicated hostname later.
- [ ] Add canonical/sitemap updates for multi-surface discovery.

## Deliverables
- Three schema-validated manifest types (`skill`, `agent`, `tool`).
- Three generated indexes in `registry/`.
- Three route modules in web app.
- No skills-only assumptions in loader and UI.

## Risk Notes
- Keep CLI backward-compatible for `skills-hub install`.
- Avoid introducing breaking changes to existing `registry/index.json` consumers until compatibility strategy is finalized.
- Use staged migration: support old index file while introducing module indexes.

## Execution Order
1. Finish Phase 2 wiring (validation scripts).
2. Implement Phase 3 registry builder outputs.
3. Implement Phase 4 web module routes.
4. Backfill Phase 5 tests.
5. Publish Phase 6 seed content.
6. Roll out Phase 7 domains.
