# Module Architecture: Skills, Agents, Tools & MCP

## Purpose
Define the target architecture for a single codebase with separate product modules and registries.

## Product Surfaces
- Skills: reusable task-level expertise packages.
- Agents: orchestrated templates that compose role, memory, skills, and tools.
- Tools & MCP: integration connectors, adapters, and MCP server definitions.

## Repository Structure
```text
skills/
  <domain>/<slug>/
agents/
  <domain>/<slug>/
tools-mcp/
  <domain>/<slug>/

registry/
  skills-index.json
  agents-index.json
  tools-index.json
```

## Route Structure
- `/skills`
- `/agents`
- `/tools-mcp`

Detail routes:
- `/skills/<domain>/<slug>`
- `/agents/<domain>/<slug>`
- `/tools-mcp/<domain>/<slug>`

## Manifest Contracts
- Skills: `skill.yaml` validated by `shared/schemas/skill.schema.json`
- Agents: `agent.yaml` validated by `shared/schemas/agent.schema.json`
- Tools: `tool.yaml` validated by `shared/schemas/tool.schema.json`

## Registry Generation
Build separate indexes from each top-level folder:
- `skills/` -> `registry/skills-index.json`
- `agents/` -> `registry/agents-index.json`
- `tools-mcp/` -> `registry/tools-index.json`

Each index entry must contain:
- `id`, `name`, `description`, `category`, `latest`, `versions`, `runtimes`, `tags`, `deprecated`
- version fields: `version`, `released_at`, `manifest_url`, `artifact_url`, `sha256`

## Domain Routing Strategy
- `skills.ai-knowledge-hub.org` serves `/skills`
- `agents.ai-knowledge-hub.org` serves `/agents`
- tools & MCP remains on `/tools-mcp` initially, with optional future domain split

## Backward Compatibility
- Maintain `registry/index.json` temporarily as an alias to skills index or aggregated view.
- Keep existing CLI install behavior for skills until dedicated agent/tool commands are defined.

## Non-Goals (Initial Rollout)
- Separate repositories
- Separate deployment pipelines per module
- Breaking changes to existing skills manifests
