# Architecture Decisions (POC)

## Scope

This document captures current decisions for turning this repository
into a public skills hub while staying on free-tier infrastructure.

## Decision 1: Repository Architecture

- Decision: use a modular monorepo structure.
- Reason:
  - shared schemas can be versioned once and reused by web + CLI
  - simpler cross-project change coordination at early stage
  - lower operational overhead for a small core team
- Initial modules:
  - `apps/web`
  - `apps/cli`
  - `packages/schemas`
  - `packages/registry-builder`

## Decision 2: Frontend and Hosting

- Decision: Next.js on Vercel for the website MVP.
- Reason:
  - fast setup and deploy flow
  - strong free tier for POC traffic
  - easy custom domain integration

## Decision 3: Backend for MVP

- Decision: no dedicated backend for v1 POC.
- Reason:
  - registry can be generated as static JSON
  - contribution flow is GitHub PR based
  - avoids unnecessary infrastructure before product validation

## Decision 4: Database Timing

- Decision: defer database choice until backend is required.
- Trigger to revisit:
  - user accounts and private data
  - dynamic usage analytics or personalization
  - write-heavy workflows not suitable for static content
- Preferred default when needed: Postgres (Neon free tier).

## Decision 5: Authentication Timing

- Decision: no authentication for initial public catalog.
- Reason:
  - contributions happen through GitHub PRs
  - public read-only catalog does not require sessions
- Trigger to revisit:
  - in-product submissions, ratings, or saved collections
- First auth option when needed: Auth.js (NextAuth) + GitHub OAuth.

## Decision 6: Registry Contract First

- Decision: define and enforce schemas before feature expansion.
- Artifacts:
  - `shared/schemas/skill.schema.json`
  - `shared/schemas/registry-index.schema.json`
- Reason:
  - CLI, website, and CI can rely on a single source of truth
  - reduces migration pain during rapid iteration

## Decision 7: Technology Stack (POC)

- Decision: keep core implementation in Go and Next.js.
- Scope:
  - Go:
    - CLI (`cmd/skills-hub`)
    - registry builder (`cmd/registry-builder`)
    - shared validation logic where practical
  - Next.js:
    - public hub website
    - static catalog rendering from registry JSON
- Reason:
  - one systems language for CLI and build tooling
  - fast web iteration and hosting fit on Vercel
  - fewer moving parts for a small team

## Decision 8: Deployment Strategy (POC)

- Decision: use static-first deployment with GitHub Actions + Vercel.
- Strategy:
  - Source of truth: this repository.
  - CI on pull request:
    - skill structure checks
    - deterministic script checks
    - schema checks
    - registry generation freshness check
  - Merge to `main`:
    - generate and commit `registry/index.json`
    - deploy website on Vercel with custom domain
- Environment model:
  - `dev` branch: staging and preview validation
  - `main` branch: production release path
- Artifact distribution:
  - short term: registry metadata + local install paths
  - next phase: hosted versioned artifacts and signatures

## Vercel Deployment Notes (Web MVP)

- Deploy target: `apps/web`.
- Build command: `npm run build`.
- Install command: `npm install`.
- Output: standard Next.js deployment on Vercel.
- Environment assumptions:
  - app reads `registry/index.json` from repository at build/runtime
  - no database or auth dependency required for MVP
- Domain plan:
  - production custom domain on Vercel
  - preview deployments from pull requests

## Open Questions

- Final public name for the hub website.
- Whether to split monorepo into multiple repos after beta.
- Whether verified badge issuance should require signed artifacts.
