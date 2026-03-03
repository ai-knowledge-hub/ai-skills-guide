# AI Skills Guide

Practical, reusable AI skills for marketing practitioners and ad-tech
software engineers, plus agent templates, tools/MCP definitions, a hub UI,
and QA automation flows.

## What this repo is

This repository is the executable companion to our written guide.
It contains production-oriented `SKILL.md` packages, deterministic
scripts, test prompts, and contribution standards.

## Positioning

AI Knowledge Hub is an open, runtime-agnostic platform for marketing and
adtech teams. We publish reusable building blocks across three modules:

- skills (task-level expertise)
- agents (orchestrated templates)
- tools & MCP connectors (integration layer)

- Guide article site:
  [ai-news-hub.performics-labs.com](https://ai-news-hub.performics-labs.com)
  (article title: The Agent Architect’s Playbook: Building AI Skills
  for Marketing & Ad Tech)
- Community references:
  [awesome-agent-skills](https://github.com/VoltAgent/awesome-agent-skills)

## Current Scope

### Skills (18)

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
15. `weekly-performance-review-bi` (BI Reporting)
16. `dashboard-generator` (BI Dashboard Build)
17. `dashboard-qa-checker` (BI QA)
18. `executive-narrative-writer` (BI Insights Communication)

### Agents (3)

1. `marketing/weekly-performance-supervisor`
2. `marketing/campaign-qa-supervisor`
3. `adtech/bi-insights-orchestrator`

### Tools & MCP (3)

1. `analytics/ga4-mcp-connector`
2. `ads/meta-ads-mcp-connector`
3. `warehouse/bigquery-mcp-query-runner`

## Recent Additions

- `ai-output-eval-scorecard`
- `cross-channel-budget-pacing-agent`
- `ab-test-planner-analyzer`
- `lifecycle-journey-trigger-designer`
- `dynamic-creative-rules-engine`
- `brand-rag-memory-bootstrap`
- `weekly-performance-review-bi`
- `dashboard-generator`
- `dashboard-qa-checker`
- `executive-narrative-writer`
- `marketing/weekly-performance-supervisor`
- `marketing/campaign-qa-supervisor`
- `adtech/bi-insights-orchestrator`
- `analytics/ga4-mcp-connector`
- `ads/meta-ads-mcp-connector`
- `warehouse/bigquery-mcp-query-runner`

## Definition of done for each module entry

- Has a module spec file:
  - skills: `SKILL.md`
  - agents: `AGENT.md`
  - tools-mcp: `TOOL.md`
- Has `tests/test-prompts.md` (>= 5 realistic prompts)
- Has `examples/` with sample input/output shape
- Has a valid manifest:
  - skills: `skill.yaml`
  - agents: `agent.yaml`
  - tools-mcp: `tool.yaml`
- Documents runtime assumptions and dependencies
- Uses scripts/config for deterministic logic where relevant

## Repository layout

```text
skills/
  marketing/
  adtech/
agents/
  marketing/
  adtech/
tools-mcp/
  analytics/
  ads/
  warehouse/
registry/
  index.json           # compatibility skills index
  skills-index.json
  agents-index.json
  tools-index.json
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

1. Pick a module entry under `skills/`, `agents/`, or `tools-mcp/`.
2. Read `README.md` and the module spec file (`SKILL.md`/`AGENT.md`/`TOOL.md`).
3. Run prompts in `tests/test-prompts.md`.
4. Verify structure with `bash scripts/validate-skills.sh`.
5. Validate manifests with `bash scripts/validate-manifests.sh`.
6. Submit improvements via PR.

## Use Agent Packages (Step-by-step)

This follows the agent model from our article:

`Agent = Role + Memory + Tools + Skills + Model`

### 1. Pick an agent package

Example:

- `marketing/weekly-performance-supervisor`
- `adtech/bi-insights-orchestrator`

Inspect details:

```bash
./bin/skills-hub info \
  --module agents \
  --entry marketing/weekly-performance-supervisor@latest
```

### 2. Install the agent and required packages

Install the agent:

```bash
./bin/skills-hub install \
  --module agents \
  --entry marketing/weekly-performance-supervisor@latest \
  --runtime codex
```

Install dependent skills:

```bash
./bin/skills-hub install \
  adtech/dashboard-generator@latest \
  --runtime codex
./bin/skills-hub install \
  adtech/dashboard-qa-checker@latest \
  --runtime codex
./bin/skills-hub install \
  adtech/executive-narrative-writer@latest \
  --runtime codex
```

Install tools/MCP connectors:

```bash
./bin/skills-hub install \
  --module tools \
  --entry analytics/ga4-mcp-connector@latest \
  --runtime codex
./bin/skills-hub install \
  --module tools \
  --entry warehouse/bigquery-mcp-query-runner@latest \
  --runtime codex
```

### 3. Configure Role and Memory

Use project-level instructions:

- role and goals
- constraints (read-only, approval gates)
- response style
- memory references (brand docs, KPI dictionary, runbooks)

For Codex, use `AGENTS.md` in repo root.
For Claude projects, use your project guidance file/workspace instructions.

### 4. Connect Tools via MCP

Map your installed tool connectors to real MCP servers and credentials:

- GA4
- warehouse (BigQuery/Redshift)
- ads platforms
- Slack/Teams for alerts

Keep tool permissions scoped. Start read-only, then expand.

### 5. Run and validate

Run your workflow request and verify:

- deterministic section ordering
- QA block behavior on critical failures
- explicit evidence values in failures
- no fabricated metrics

### Runtime examples

#### A) Claude Code / Claude Agent SDK

Docs:

- [Claude API Docs](https://platform.claude.com/docs/en/home)

Install to Claude runtime paths:

```bash
./bin/skills-hub install \
  --module agents \
  --entry marketing/weekly-performance-supervisor@latest \
  --runtime claude
./bin/skills-hub install \
  adtech/dashboard-generator@latest \
  --runtime claude
./bin/skills-hub install \
  --module tools \
  --entry analytics/ga4-mcp-connector@latest \
  --runtime claude
```

Then map MCP servers in your Claude environment and add project constraints
for approval gates before live actions.

#### B) OpenAI Codex

Docs:

- [OpenAI Codex Docs](https://developers.openai.com/codex/)

Install to Codex runtime paths:

```bash
./bin/skills-hub install \
  --module agents \
  --entry adtech/bi-insights-orchestrator@latest \
  --runtime codex
./bin/skills-hub install \
  adtech/analyst-copilot-bigquery-redshift@latest \
  --runtime codex
./bin/skills-hub install \
  --module tools \
  --entry warehouse/bigquery-mcp-query-runner@latest \
  --runtime codex
```

Add an `AGENTS.md` with role, goals, boundaries, and preferred tools.

#### C) OpenClaw or other generic agent runtimes

Source:

- [OpenClaw](https://github.com/openclaw/openclaw)

Install with explicit targets:

```bash
./bin/skills-hub install \
  --module agents \
  --entry marketing/weekly-performance-supervisor@latest \
  --runtime generic \
  --target ./openclaw-workspace/agents
./bin/skills-hub install \
  adtech/dashboard-generator@latest \
  --runtime generic \
  --target ./openclaw-workspace/skills
./bin/skills-hub install \
  --module tools \
  --entry analytics/ga4-mcp-connector@latest \
  --runtime generic \
  --target ./openclaw-workspace/tools-mcp
```

Mount those folders into your runtime workspace and reference them in your
agent onboarding/config flow.

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

Generate module indexes from manifests:

```bash
make registry
```

This now writes:

- `registry/skills-index.json`
- `registry/agents-index.json`
- `registry/tools-index.json`
- `registry/index.json` (skills compatibility index)

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
./bin/skills-hub info \
  --module agents \
  --entry adtech/bi-insights-orchestrator@latest
./bin/skills-hub install \
  --module agents \
  --entry marketing/weekly-performance-supervisor@latest \
  --runtime codex
./bin/skills-hub install \
  --module tools \
  --entry analytics/ga4-mcp-connector@latest \
  --runtime generic \
  --target ./my-agent/tools-mcp
./bin/skills-hub run-agent \
  --agent marketing/weekly-performance-supervisor \
  --bindings agents/marketing/weekly-performance-supervisor/config/\
tool-bindings.example.json \
  --memory agents/marketing/weekly-performance-supervisor/config/\
memory-profile.example.json \
  --governance agents/marketing/weekly-performance-supervisor/config/\
governance.example.json \
  --approve-live \
  --audit-log ./tmp/weekly-performance-supervisor-run.json
```

Runtime target defaults:

- `--runtime codex`:
  - skills -> `$CODEX_HOME/skills` (or `~/.codex/skills`)
  - agents -> `$CODEX_HOME/agents` (or `~/.codex/agents`)
  - tools -> `$CODEX_HOME/tools-mcp` (or `~/.codex/tools-mcp`)
- `--runtime claude`:
  - skills -> `$CLAUDE_HOME/skills` (or `$CLAUDE_CODE_HOME/skills`,
    or `~/.claude/skills`)
  - agents -> `$CLAUDE_HOME/agents` (or `$CLAUDE_CODE_HOME/agents`,
    or `~/.claude/agents`)
  - tools -> `$CLAUDE_HOME/tools-mcp` (or `$CLAUDE_CODE_HOME/tools-mcp`,
    or `~/.claude/tools-mcp`)
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
- `/agents` searchable catalog
- `/tools-mcp` searchable catalog
- `/skills/<category>/<slug>` skill details and install snippets
- `/agents/<category>/<slug>` agent details
- `/tools-mcp/<category>/<slug>` tool/MCP details

Smoke E2E tests:

```bash
cd apps/web
pnpm test:e2e:setup
pnpm test:e2e
```

Static manifest/artifact URLs:

- During web build, `pnpm prepare:assets` generates static files under
  `apps/web/public`.
- This makes registry `manifest_url` and `artifact_url` resolvable on
  production routes like:
  - `/skills/.../skill.yaml`
  - `/agents/.../agent.yaml`
  - `/tools-mcp/.../tool.yaml`
  - `/artifacts/<id>/<version>.tar.gz`
