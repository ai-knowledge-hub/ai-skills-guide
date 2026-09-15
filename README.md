# AI Skills Guide

Practical, reusable AI skills for marketing practitioners, software
engineers, security teams, and agent builders, plus agent templates,
tools/MCP definitions, a hub UI, and QA automation flows.

## What this repo is

This repository is the practical companion to our written guide. It contains
instruction packages, local executables, integration specifications,
orchestration templates, plugin source compositions, and documentation-only
packs. Every registry entry declares its usability classification explicitly.

## Positioning

AI Knowledge Hub is an open, runtime-agnostic platform for applied agent
workflows. The catalog started with marketing and adtech use cases and now
expands into engineering maintenance, cybersecurity, and agent operations.
We publish reusable building blocks across four modules:

- skills (task-level expertise)
- agents (orchestrated templates)
- plugins (composition templates today; installable bundles when implemented
  and verified)
- tools & MCP connectors (integration layer)

`packs/` contains documentation-only playbooks that curate existing catalog
entries. Packs are not installable modules.

## Know what you are installing

Review maturity and operational usability answer different questions:

- `readiness` says whether an entry is experimental, reviewed, or deprecated.
- `usability.availability` says whether it can be used now, needs setup, is a
  template, or is documentation only.
- `usability.execution` says whether it behaves as instructions, a local tool,
  remote integration, orchestrator, bundle, or documentation.

- `usable-now`: Install and use the instructions or local executable
  immediately.
- `setup-required`: Configure credentials, bindings, dependencies, or policy
  for an implemented artifact before use.
- `not-verified`: Implementation exists, but current target-scoped operational
  evidence is not recorded.
- `template-only`: Implement the supplied contract or scaffold before treating
  it as an executable integration.
- `documentation-only`: Use the package as a learning or architecture guide;
  it is not installed as a runtime capability.

Operational lifecycle, evidence, promotion, demotion, and verification expiry
are governed by the normative
[Operational Readiness and Promotion Contract](shared/contracts/operational-readiness-v1.md).
In particular, prompt tests and schema-valid manifests do not by themselves
prove that an entry is operationally available.

Registry admission rejects manifests that omit `usability.availability` or
`usability.execution`; registry generation preserves those declarations and
does not infer them from module names or directories. Current catalog manifests
use schema `1.1`, and generated indexes use registry format `1.3`; consumers
must fail closed on unsupported versions rather than reinterpret new states.

See [docs/using-the-catalog.md](docs/using-the-catalog.md) for module behavior,
install effects, and first-run guidance.

## Catalog scope

Generated registry files are the source of truth for current catalog contents:

- `registry/skills-index.json`
- `registry/agents-index.json`
- `registry/tools-index.json`
- `registry/plugins-index.json`

Use `./bin/skills-hub list --module <skills|agents|tools|plugins>` for the
current inventory. `packs/` remains documentation-only and is intentionally
outside the install registry.

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
- `plugins/marketing/*`: source-composition templates that reference existing
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

The default `local` source is an explicit development mode. It resolves a
repository index and copies files from the current checkout. To install a
published release without cloning this repository, select the `remote` source:

```bash
./bin/skills-hub install \
  --source remote \
  --registry-url \
    https://skills.ai-knowledge-hub.org/registry/skills-index.json \
  --entry engineering/implementation-strategy@latest \
  --runtime codex
```

Remote mode accepts only HTTPS registry and artifact URLs. It verifies the
downloaded archive against the registry SHA-256, validates its paths and
manifest, and stages the complete package before changing the runtime. Use
`--offline` after one successful online fetch to reuse verified cached bytes.
See [Remote package installation](docs/remote-install.md) for source modes,
cache behavior, trust boundaries, and recovery guidance.

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

## Use Plugin Templates

Current plugin entries are `template-only` source compositions. The operational
`skills-hub install --module plugins` path rejects them before writing runtime
files, resolving dependencies, or generating Codex/Claude registration
manifests. Review each plugin's `README.md`, `plugin.json`, referenced modules,
hooks, secrets, and approval rules in the repository, then implement and verify
a self-contained bundle before changing its availability.

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
  - `/release-manifests/<module>/<id>/<version>.yaml`
