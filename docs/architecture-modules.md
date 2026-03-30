# Module Architecture: Skills, Agents, Plugins, Tools & MCP

## Purpose
Define the target architecture for a single codebase with separate product
modules and registries.

## Product Surfaces
- Skills: reusable task-level expertise packages across marketing, adtech,
  engineering, security, and agent operations.
- Agents: orchestrated templates that compose role, memory, skills, and tools.
- Plugins: installable bundles that package skills, agents, tools, hooks, and
  setup guidance into portable team capabilities.
- Tools & MCP: integration connectors, adapters, and MCP server definitions.

## Repository Structure
```text
skills/
  <domain>/<slug>/
agents/
  <domain>/<slug>/
tools-mcp/
  <domain>/<slug>/
plugins/
  <domain>/<slug>/

registry/
  skills-index.json
  agents-index.json
  plugins-index.json
  tools-index.json
```

## Route Structure
- `/skills`
- `/agents`
- `/plugins`
- `/tools-mcp`

Detail routes:
- `/skills/<domain>/<slug>`
- `/agents/<domain>/<slug>`
- `/plugins/<domain>/<slug>`
- `/tools-mcp/<domain>/<slug>`

## Manifest Contracts
- Skills: `skill.yaml` validated by `shared/schemas/skill.schema.json`
- Agents: `agent.yaml` validated by `shared/schemas/agent.schema.json`
- Plugins: `plugin.yaml` validated by `shared/schemas/plugin.schema.json`
- Tools: `tool.yaml` validated by `shared/schemas/tool.schema.json`

## Operational Metadata Rollout

The catalog is moving toward a shared operational metadata model so detail
pages answer practical questions quickly:

- what this entry touches
- what permissions it needs
- what trust boundary it crosses
- what approval boundary applies

The first implemented slices are `tools-mcp` and `agents`.

`tools-mcp` now documents:

- `connected_system`
- `capabilities`
- `auth_required`
- `access_level`
- `trust_boundary`
- `approval_boundary`

`agents` now documents:

- `role`
- `coordinates`
- `autonomy_level`
- `approval_boundary`
- `outputs`

`skills` now documents:

- `use_when`
- `execution_mode`
- `outputs`
- `approval_boundary`

## Registry Generation
Build separate indexes from each top-level folder:
- `skills/` -> `registry/skills-index.json`
- `agents/` -> `registry/agents-index.json`
- `plugins/` -> `registry/plugins-index.json`
- `tools-mcp/` -> `registry/tools-index.json`

Each index entry must contain:
- `id`, `name`, `description`, `category`, `latest`, `versions`, `runtimes`, `tags`, `deprecated`
- version fields: `version`, `released_at`, `manifest_url`, `artifact_url`, `sha256`

## Domain Routing Strategy
- `skills.ai-knowledge-hub.org` serves `/skills`
- `agents.ai-knowledge-hub.org` serves `/agents`
- plugins remain on `/plugins` initially, with optional future domain split
- tools & MCP remains on `/tools-mcp` initially, with optional future domain split

## Backward Compatibility
- Maintain `registry/index.json` temporarily as an alias to skills index or aggregated view.
- Keep existing CLI install behavior for skills until dedicated agent/tool commands are defined.

## Non-Goals (Initial Rollout)
- Separate repositories
- Separate deployment pipelines per module
- Breaking changes to existing skills manifests

## Current Expansion Areas

- `engineering/*` skills for code maintenance and verification
- `security/*` skills for prompt safety and supply-chain defense
- `agentops/*` skills for harness reflection and controlled evolution
