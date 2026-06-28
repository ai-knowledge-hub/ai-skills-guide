# AI Skills Guide

Practical, reusable AI skills for marketing practitioners, software
engineers, security teams, and agent builders, plus agent templates,
tools/MCP definitions, a hub UI, and QA automation flows.

## What this repo is

This repository is the executable companion to our written guide.
It contains production-oriented `SKILL.md` packages, deterministic
scripts, test prompts, and contribution standards.

## Positioning

AI Knowledge Hub is an open, runtime-agnostic platform for applied agent
workflows. The catalog started with marketing and adtech use cases and now
expands into engineering maintenance, cybersecurity, and agent operations.
We publish reusable building blocks across four modules:

- skills (task-level expertise)
- agents (orchestrated templates)
- plugins (installable composition layer)
- tools & MCP connectors (integration layer)

`packs/` contains documentation-only playbooks that curate existing catalog
entries. Packs are not installable modules.

## Operational Metadata Standard

The catalog is being normalized around a shared operational metadata model so
users can answer the same practical questions across modules:

- what this entry does
- what systems it touches
- what it needs to authenticate
- what permissions it can exercise
- what approval boundary applies

The first rollouts are on `tools-mcp` and `agents`.

`tools-mcp` now exposes:

- connected system
- capabilities
- auth required
- access level
- trust boundary
- approval boundary

`agents` now exposes:

- role
- coordinates
- autonomy level
- approval boundary
- outputs

`skills` now exposes:

- use when
- execution mode
- outputs
- approval boundary

- Guide article site:
  [ai-news-hub.performics-labs.com](https://ai-news-hub.performics-labs.com)
  (article title: The Agent Architect’s Playbook: Building AI Skills
  for Marketing & Ad Tech)
- Community references:
  [awesome-agent-skills](https://github.com/VoltAgent/awesome-agent-skills)

## Current Scope

### Skills (39)

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
19. `engineering/implementation-strategy` (Code Maintenance)
20. `engineering/code-change-verification` (Code Maintenance)
21. `engineering/test-gap-analyzer` (Testing Quality)
22. `engineering/coverage-gap-reporter` (Testing Quality)
23. `engineering/pr-review-and-draft` (PR Review)
24. `security/handle-untrusted-content` (Prompt Safety)
25. `security/dependency-supply-chain-audit` (Supply Chain)
26. `security/secrets-and-credential-hygiene` (Runtime Hardening)
27. `security/environment-risk-assessment` (Runtime Hardening)
28. `agentops/harness-run-reflection` (Harness Evolution)
29. `agentops/harness-skill-proposal` (Harness Governance)
30. `agentops/harness-regression-evaluator` (Harness Evaluation)
31. `agentops/agent-control-plane-review` (Harness Governance)
32. `security/marketing-agent-risk-review` (Runtime Hardening)
33. `security/ad-platform-agent-auth-review` (Runtime Hardening)
34. `agentops/ad-platform-policy-gate-designer` (Harness Governance)
35. `marketing/creative-operating-system-audit` (Creative Ops / Anti-Slop)
36. `marketing/utility-campaign-concept-designer` (Creative Ops / Utility)
37. `marketing/product-as-media-mapper` (Creative Ops / Owned Surfaces)
38. `marketing/cultural-timing-signal-triage` (Creative Ops / Cultural Timing)
39. `marketing/creator-strategy-brief` (Creator Ops)

### Agents (6)

1. `marketing/weekly-performance-supervisor`
2. `marketing/campaign-qa-supervisor`
3. `adtech/bi-insights-orchestrator`
4. `agentops/agent-control-plane-supervisor`
5. `adtech/ad-platform-control-plane-supervisor`
6. `marketing/creative-operating-system-supervisor`

### Tools & MCP (5)

1. `analytics/ga4-mcp-connector`
2. `ads/meta-ads-mcp-connector`
3. `warehouse/bigquery-mcp-query-runner`
4. `agentops/agent-control-plane-server`
5. `adtech/ad-platform-executor-template`

### Plugins (10)

1. `marketing/performance-reporting-plugin`
2. `marketing/campaign-audit-plugin`
3. `marketing/content-repurposing-plugin`
4. `marketing/page-speed-technical-seo-plugin`
5. `marketing/competitive-intelligence-plugin`
6. `marketing/ad-creative-plugin`
7. `engineering/code-maintenance-plugin`
8. `security/runtime-safety-plugin`
9. `agentops/harness-governance-plugin`
10. `marketing/creative-operating-system-plugin`

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
- `marketing/performance-reporting-plugin`
- `marketing/campaign-audit-plugin`
- `marketing/content-repurposing-plugin`
- `marketing/page-speed-technical-seo-plugin`
- `marketing/competitive-intelligence-plugin`
- `marketing/ad-creative-plugin`
- `engineering/code-maintenance-plugin`
- `security/runtime-safety-plugin`
- `agentops/harness-governance-plugin`
- `engineering/implementation-strategy`
- `engineering/code-change-verification`
- `engineering/test-gap-analyzer`
- `engineering/coverage-gap-reporter`
- `engineering/pr-review-and-draft`
- `security/handle-untrusted-content`
- `security/dependency-supply-chain-audit`
- `security/secrets-and-credential-hygiene`
- `security/environment-risk-assessment`
- `agentops/harness-run-reflection`
- `agentops/harness-skill-proposal`
- `agentops/harness-regression-evaluator`
- `agentops/agent-control-plane-review`
- `security/marketing-agent-risk-review`
- `agentops/agent-control-plane-supervisor`
- `agentops/agent-control-plane-server`
- `security/ad-platform-agent-auth-review`
- `agentops/ad-platform-policy-gate-designer`
- `adtech/ad-platform-control-plane-supervisor`
- `adtech/ad-platform-executor-template`
- `marketing/creative-operating-system-audit`
- `marketing/utility-campaign-concept-designer`
- `marketing/product-as-media-mapper`
- `marketing/cultural-timing-signal-triage`
- `marketing/creator-strategy-brief`
- `marketing/creative-operating-system-supervisor`
- `marketing/creative-operating-system-plugin`

## Definition of done for each module entry

- Has a module spec file:
  - skills: `SKILL.md`
  - agents: `AGENT.md`
  - plugins: `plugin.json`
  - tools-mcp: `TOOL.md`
- Has `tests/test-prompts.md` (>= 5 realistic prompts)
- Has `examples/` with sample input/output shape
- Has a valid manifest:
  - skills: `skill.yaml`
  - agents: `agent.yaml`
  - plugins: `plugin.yaml`
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
plugins/
  marketing/
tools-mcp/
  analytics/
  ads/
  warehouse/
registry/
  index.json           # compatibility skills index
  skills-index.json
  agents-index.json
  tools-index.json
packs/
  creative-operating-system/
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

## New Skill Packs

- `skills/engineering/*`: code maintenance, verification, tests, coverage,
  and PR review workflows.
- `skills/security/*`: prompt safety, dependency audit, secrets hygiene,
  and runtime risk assessment.
- `skills/agentops/*`: harness reflection, skill proposal, and regression
  evaluation for self-evolving agent scaffolds.
- `plugins/marketing/*`: installable bundles that package existing
  skills, agents, tools, hooks, and setup guidance.

## Quickstart

1. Pick a module entry under `skills/`, `agents/`, `plugins/`, or `tools-mcp/`.
2. Read `README.md` and the module spec file
   (`SKILL.md`/`AGENT.md`/`plugin.json`/`TOOL.md`).
3. Run prompts in `tests/test-prompts.md`.
4. Verify structure with `bash scripts/validate-skills.sh`.
5. Validate manifests with `bash scripts/validate-manifests.sh`.
6. Submit improvements via PR.

Release workflow and version bump definitions are documented in
[docs/versioning-and-release.md](docs/versioning-and-release.md).

## Install New Skill Packs

For Codex:

```bash
./bin/skills-hub install engineering/implementation-strategy@latest \
  --runtime codex
./bin/skills-hub install security/handle-untrusted-content@latest \
  --runtime codex
./bin/skills-hub install agentops/harness-run-reflection@latest \
  --runtime codex
```

For Claude:

```bash
./bin/skills-hub install engineering/code-change-verification@latest \
  --runtime claude
./bin/skills-hub install security/dependency-supply-chain-audit@latest \
  --runtime claude
./bin/skills-hub install agentops/harness-regression-evaluator@latest \
  --runtime claude
```

For generic runtimes:

```bash
./bin/skills-hub install engineering/pr-review-and-draft@latest \
  --runtime generic \
  --target ./my-agent/skills
./bin/skills-hub install security/environment-risk-assessment@latest \
  --runtime generic \
  --target ./my-agent/skills
./bin/skills-hub install agentops/harness-skill-proposal@latest \
  --runtime generic \
  --target ./my-agent/skills
```

## Install Plugins

Plugins bundle existing skills, agents, tools, hooks, and setup guidance into
one installable package. For `codex` and `claude`, the installer also generates
a runtime-specific manifest inside the installed plugin directory:

- `codex` -> `.codex-plugin/plugin.json`
- `claude` -> `.claude-plugin/plugin.json`

Plugin installs also resolve bundled dependencies automatically:

- referenced `skills` install into the runtime skills directory
- referenced `agents` install into the runtime agents directory
- referenced `tools-mcp` install into the runtime tools directory
- packaged `hooks/` remain inside the installed plugin directory

For Codex:

```bash
./bin/skills-hub install --module plugins \
  --entry marketing/performance-reporting-plugin@latest \
  --runtime codex
```

Expected result:

- plugin files copied into your Codex plugins directory
- generated `.codex-plugin/plugin.json`
- bundled skills, agents, and tools installed into sibling Codex runtime directories
- CLI output listing bundled component IDs, required secrets, and approvals

For Claude:

```bash
./bin/skills-hub install --module plugins \
  --entry marketing/competitive-intelligence-plugin@latest \
  --runtime claude
```

Expected result:

- plugin files copied into your Claude plugins directory
- generated `.claude-plugin/plugin.json`
- bundled skills, agents, and tools installed into sibling Claude runtime directories
- CLI output warning when the plugin is not security reviewed

For generic runtimes:

```bash
./bin/skills-hub install --module plugins \
  --entry marketing/ad-creative-plugin@latest \
  --runtime generic \
  --target ./my-agent/plugins
```

Expected result:

- plugin files copied into `./my-agent/plugins`
- bundled skills, agents, and tools installed into sibling directories under `./my-agent/`
- no runtime-specific manifest generated automatically
- you wire the plugin into your runtime manually

Before enabling any plugin in a live environment:

- review bundled skills, agents, and tool references
- confirm required secrets are scoped correctly
- confirm approval rules are compatible with your runtime
- inspect generated runtime manifests before activation
- treat `security_reviewed: false` as review-required, not install-ready

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

- [OpenAI Codex Docs](https://developers.openai.com/codex)

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
For security expectations, see `SECURITY_BASELINE.md`.
For new pack guidance, see `docs/skill-pack-code-maintenance.md`,
`docs/skill-pack-cybersecurity.md`, and
`docs/skill-pack-autoharnessing.md`.
For plugin guidance, see `docs/plugin-architecture.md`,
`docs/plugin-authoring-guide.md`, and
`docs/plugin-security-and-review.md`.

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
- `/plugins` searchable catalog
- `/tools-mcp` searchable catalog
- `/skills/<category>/<slug>` skill details and install snippets
- `/agents/<category>/<slug>` agent details
- `/plugins/<category>/<slug>` plugin details
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
  - `/plugins/.../plugin.yaml`
  - `/tools-mcp/.../tool.yaml`
  - `/artifacts/<id>/<version>.tar.gz`
