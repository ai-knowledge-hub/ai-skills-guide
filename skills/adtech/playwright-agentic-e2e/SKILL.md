---
name: playwright-agentic-e2e
description: Plan, generate, run, and heal Playwright smoke tests for web routes and core UI interactions in monorepo-safe setups.
---

# Playwright Agentic E2E QA

## When to use
Use for repeatable browser QA before merge, especially when UI changes often and tests need fast regeneration/healing.

## Inputs required
- app_base_url or local dev command
- target routes (default smoke set)
- selectors or user-visible labels for key actions
- monorepo path hints (where `playwright.config.*` and app package live)

## Workflow
1. **Planner**: lock test scope first (smoke routes + highest-risk interactions).
2. **Generator**: produce Playwright tests using user-facing locators and stable assertions.
3. **Runner**: execute tests locally/CI and capture traces/screenshots on failure.
4. **Healer**: when tests break, update selectors/flows with smallest safe patch and re-run.
5. **Promote**: keep the suite reusable as shared QA coverage for new hub features.

## Monorepo / subdir setup rules
- Always set explicit `testDir`, `outputDir`, and `webServer.cwd` in Playwright config.
- Keep CI working directory explicit when app code is in a subfolder (for example `apps/web`).
- Prefer a deterministic base URL (`http://127.0.0.1:3000`) for local + CI parity.

## Output format
- Test Plan: routes + interactions + pass criteria
- Generated/updated test files
- Run Summary: pass/fail, key failures, healing changes applied
- Follow-up Risks: flaky areas and selector hardening actions

## Guardrails
- Do not claim coverage beyond implemented tests.
- Keep smoke scope small and deterministic before expanding.
- Never auto-delete failing tests; heal with minimal, explainable changes.
- If selectors are unstable, prefer role/text/test-id improvements over brittle CSS chains.
