---
name: playwright-vscode-loop-codex
description: Use Codex as orchestrator for Playwright planner, generator, and healer loops executed through VS Code-compatible Playwright agent flow.
---

# Playwright VS Code Loop for Codex

## When to use
Use when you want Codex to run repeatable UX smoke/regression loops while delegating Playwright agent execution to a VS Code-compatible loop.

## Inputs required
- app_dir (for monorepos, explicit app subdirectory)
- playwright_config_path
- base_url and/or local web command
- smoke routes and high-risk interactions
- sample detail page route

## Workflow
1. **Planner phase**
   - Confirm target routes and pass criteria.
   - Keep first scope to smoke tests only.
2. **Generator phase**
   - Generate or update Playwright specs with stable locators (role/text/test-id).
   - Ensure config has explicit `testDir`, `outputDir`, and `webServer.cwd`.
3. **Runner phase**
   - Run Playwright tests and collect traces/report artifacts.
4. **Healer phase**
   - Apply the smallest safe selector/flow patch for failures.
   - Re-run and summarize behavioral changes.
5. **Promote phase**
   - Keep a reusable baseline suite for all future web feature PRs.

## VS Code loop execution notes
- Codex remains the orchestrator and decision-maker.
- Use VS Code loop tooling for the Playwright agent cycle where supported.
- If loop tooling is unavailable, run equivalent planner/generator/healer steps manually with Playwright tests.

## Global Codex usage
- Keep this skill in your global Codex skills directory for cross-project reuse.
- Suggested install target: `$CODEX_HOME/skills/playwright-vscode-loop-codex`.
- In each project, pass explicit monorepo paths before generation.

## Output format
- Test Plan
- Generated/Updated Files
- Run Summary
- Healing Notes
- Residual Risk

## Guardrails
- Never claim passing coverage for untested flows.
- Do not widen scope automatically from smoke to full regression.
- Do not auto-delete failing tests; heal with minimal changes.
- Prefer robust user-facing locators over brittle CSS selectors.
