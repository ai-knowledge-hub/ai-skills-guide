# Deploy Web App on Vercel

This repo is a monorepo. The web app lives at `apps/web`.

## Vercel project setup

1. Import `ai-knowledge-hub/ai-skills-guide` in Vercel.
2. Set **Root Directory** to `apps/web`.
3. Framework preset: **Next.js**.
4. Build command: `pnpm build`.
5. Install command: `pnpm install --frozen-lockfile`.
6. Output directory: leave default for Next.js.

## Git branch environments

- **Production branch**: `main`
- **Preview branch**: `dev` (and optional feature branches)

## Domain

Attach custom domain:

- `hub.ai-knowledge-hub.org`

Then set DNS records in your DNS provider as instructed by Vercel.

## CI alignment

This repo CI verifies the same core web checks before merge:

- `pnpm install --frozen-lockfile`
- `pnpm lint`
- `pnpm build`
- `npx playwright test` (smoke)

## Local parity commands

```bash
cd apps/web
pnpm install --frozen-lockfile
pnpm lint
pnpm build
pnpm test:e2e
```
