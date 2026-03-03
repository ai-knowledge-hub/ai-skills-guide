# Playwright VS Code Loop for Codex

Use this skill when Codex orchestrates a Playwright planner/generator/
healer loop that runs through a VS Code-compatible workflow.

## What this skill does

- Plans smoke scope and acceptance checks.
- Generates and updates Playwright specs.
- Runs healing loops and reports minimal fixes.

## Before you start

1. Confirm app directory and Playwright config path.
2. Define smoke routes and core interactions.
3. Confirm the VS Code loop backend is available.

## Install

```bash
./bin/skills-hub install \
  adtech/playwright-vscode-loop-codex@latest \
  --runtime codex
```

## First run prompt

```text
Use Playwright VS Code Loop for Codex.
App dir: apps/web
Config: apps/web/playwright.config.ts
Scope: smoke tests for /skills, /agents, /tools-mcp.
Return plan, generated files, run summary, and healing notes.
```

## What good output looks like

- Planner scope is explicit.
- Generated tests are deterministic.
- Healing patches are minimal and explainable.

## Beginner safety checklist

- Keep first pass smoke-only.
- Re-run after each fix.
- Track residual flaky areas explicitly.
