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

## Open Questions

- Final public name for the hub website.
- Whether to split monorepo into multiple repos after beta.
- Whether verified badge issuance should require signed artifacts.
