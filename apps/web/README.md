# Web App (Hub MVP)

This app renders a static-first skills catalog from `../../registry/index.json`.

## Local development

```bash
cd apps/web
pnpm install
pnpm dev
```

Then open `http://localhost:3000`.

## E2E smoke tests (Playwright)

First-time setup downloads the browser binary:

```bash
cd apps/web
pnpm test:e2e:setup
```

Run tests:

```bash
pnpm test:e2e
```

Current smoke coverage:

- home module navigation
- skills catalog and skill detail
- agents catalog and agent detail
- tools-mcp catalog and tool detail

## Build

```bash
pnpm prepare:assets
pnpm build
pnpm start
```

`pnpm prepare:assets` creates deterministic package candidates and admits them
to the tracked, append-only `releases/` store. Reusing an `(id, version)` with
different manifest or archive bytes fails the build. Every build restores all
retained archives and versioned manifests into `apps/web/public`, then publishes
their artifact and manifest SHA-256 digests in registry format `1.3`. Repository
indexes retain source-tree digests for local development. Each release also
stores its immutable registry projection, so removing mutable catalog source
does not make a released version unresolvable. Asset preparation requires both
Node.js and Go because retained archives pass through the installer's bounded
extraction and admission checks before indexing.

## Routes

- `/` overview
- `/skills` catalog with filters
- `/skills/<category>/<slug>` skill details with install snippets
- `/agents` catalog with filters
- `/agents/<category>/<slug>` agent details
- `/tools-mcp` catalog with filters
- `/tools-mcp/<category>/<slug>` tool and MCP details
