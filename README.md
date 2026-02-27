# AI Skills Guide

Practical, reusable AI skills for marketing practitioners and ad-tech
software engineers, plus a hub UI and QA automation flows.

## What this repo is

This repository is the executable companion to our written guide.
It contains production-oriented `SKILL.md` packages, deterministic
scripts, test prompts, and contribution standards.

## Positioning

AI Knowledge Hub is an open, runtime-agnostic skills platform for
marketing and adtech teams. We publish reusable AI skill packages with
guardrails, tests, and install paths so teams can stop rebuilding the
same automations in silos.

- Guide article site:
  [ai-news-hub.performics-labs.com](https://ai-news-hub.performics-labs.com)
  (article title: The Agent Architect’s Playbook: Building AI Skills
  for Marketing & Ad Tech)
- Community references:
  [awesome-agent-skills](https://github.com/VoltAgent/awesome-agent-skills)

## Current Scope (14 practical skills)

1. `meta-google-weekly-performance-review` (Beginner)
2. `creative-workshop-pmax-reels` (Intermediate)
3. `lifecycle-experiment-planner` (Intermediate)
4. `policy-brand-compliance-checker` (Intermediate)
5. `seo-paid-search-synergy` (Advanced)
6. `analyst-copilot-bigquery-redshift` (Advanced)
7. `playwright-agentic-e2e` (QA / Infrastructure)
8. `playwright-vscode-loop-codex` (QA / VS Code Loop)
9. `ai-output-eval-scorecard` (Governance / Evaluation)
10. `cross-channel-budget-pacing-agent` (Ads Ops)
11. `ab-test-planner-analyzer` (Measurement / Experimentation)
12. `lifecycle-journey-trigger-designer` (Lifecycle CRM)
13. `dynamic-creative-rules-engine` (Creative Ops / Personalization)
14. `brand-rag-memory-bootstrap` (Analytics Engineering / RAG)

## New in v0.2 alpha

- `ai-output-eval-scorecard`
- `cross-channel-budget-pacing-agent`
- `ab-test-planner-analyzer`
- `lifecycle-journey-trigger-designer`
- `dynamic-creative-rules-engine`
- `brand-rag-memory-bootstrap`

## Definition of done for each skill

- Has `SKILL.md` with clear routing intent and guardrails
- Has `tests/test-prompts.md` (>= 5 realistic prompts)
- Has `examples/` with sample input/output shape
- Documents runtime assumptions and dependencies
- Uses scripts/config for deterministic logic where relevant

## Repository layout

```text
skills/
  marketing/
  adtech/
apps/
  web/
shared/
  metrics/
  policies/
  schemas/
  naming/
docs/
scripts/
.github/
```

## Provider and framework examples to explore

Cross-runtime examples are cataloged in
[awesome-agent-skills](https://github.com/VoltAgent/awesome-agent-skills).
Useful sections include skills for:

- OpenAI Codex / Agent Skills
- Claude-style skills
- Gemini CLI patterns
- GitHub Copilot / VS Code patterns
- Vercel AI SDK agent resources

## Quickstart

1. Pick a skill folder under `skills/`.
2. Read `README.md` + `SKILL.md` for required inputs.
3. Run prompts in `tests/test-prompts.md`.
4. Verify structure with `bash scripts/validate-skills.sh`.
5. Submit improvements via PR.

## CLI Scaffold (Go)

This repo now includes a starter CLI at `cmd/skills-hub`
for local skill management.

Build and test:

```bash
make cli-test
make cli-build
```

Schema validation (requires `check-jsonschema`):

```bash
python3 -m pip install check-jsonschema
make manifests
```

Generate `registry/index.json` from manifests:

```bash
make registry
```

Example usage:

```bash
./bin/skills-hub list
./bin/skills-hub search --tag paid-media --runtime codex
./bin/skills-hub validate
./bin/skills-hub info \
  --skill marketing/meta-google-weekly-performance-review@latest
./bin/skills-hub install \
  marketing/meta-google-weekly-performance-review@latest \
  --runtime codex
./bin/skills-hub install \
  marketing/meta-google-weekly-performance-review@latest \
  --runtime claude
./bin/skills-hub install \
  marketing/meta-google-weekly-performance-review@0.1.0 \
  --runtime generic \
  --target ./my-agent/skills
```

Runtime target defaults:

- `--runtime codex` -> `$CODEX_HOME/skills` (or `~/.codex/skills`)
- `--runtime claude` -> `$CLAUDE_HOME/skills` (or `$CLAUDE_CODE_HOME/skills`,
  or `~/.claude/skills`)
- `--runtime generic` -> requires explicit `--target`

## Contributing

See `CONTRIBUTING.md` and `docs/how-to-contribute-a-skill.md`.
For local toolchain setup, see `docs/dev-setup.md`.

## Hub Website (MVP Scaffold)

The repo now includes a Next.js catalog app at `apps/web`.

Production site:
[skills.ai-knowledge-hub.org](https://skills.ai-knowledge-hub.org/)

```bash
cd apps/web
pnpm install
pnpm dev
```

Core routes:

- `/` overview
- `/skills` searchable catalog
- `/skills/<category>/<slug>` skill details and install snippets

Smoke E2E tests:

```bash
cd apps/web
pnpm test:e2e:setup
pnpm test:e2e
```
