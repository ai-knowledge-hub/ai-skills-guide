# Playwright Agentic E2E QA

Use this skill to plan, generate, run, and heal Playwright smoke tests
for web apps.

## What this skill does

- Defines smoke scope and pass criteria.
- Generates Playwright tests with stable locators.
- Runs tests and applies minimal healing patches.

## Before you start

1. Confirm app URL or local dev command.
2. List high-risk routes/interactions.
3. Confirm Playwright config path.

## Install

```bash
./bin/skills-hub install \
  adtech/playwright-agentic-e2e@latest \
  --runtime codex
```

## First run prompt

```text
Use Playwright Agentic E2E QA.
Target routes: /, /skills, /agents, /tools-mcp
Generate smoke tests, run them, and summarize failures
with minimal healing suggestions.
```

## What good output looks like

- Scope is small and explicit.
- Assertions use user-facing locators.
- Failures include actionable root cause notes.

## Beginner safety checklist

- Do not claim coverage for untested flows.
- Do not auto-delete failing tests.
- Prefer robust locators over brittle CSS selectors.
