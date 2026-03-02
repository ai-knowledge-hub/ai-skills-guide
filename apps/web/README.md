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
pnpm build
pnpm start
```

## Routes

- `/` overview
- `/skills` catalog with filters
- `/skills/<category>/<slug>` skill details with install snippets
- `/agents` catalog with filters
- `/agents/<category>/<slug>` agent details
- `/tools-mcp` catalog with filters
- `/tools-mcp/<category>/<slug>` tool and MCP details
