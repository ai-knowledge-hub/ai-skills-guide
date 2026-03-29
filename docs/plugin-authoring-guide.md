# Plugin Authoring Guide

## When to create a plugin

Create a plugin when a workflow needs to be shared as one installable team
capability instead of as separate skills, agents, and tools.

Good plugin candidates:
- weekly reporting bundles
- campaign QA bundles
- SEO diagnostic bundles
- content repurposing bundles

## Minimum required files

```text
<plugin>/
  README.md
  plugin.yaml
  plugin.json
  examples/
  tests/
    test-prompts.md
```

## Authoring rules

1. Keep plugins compositional.
   Prefer referencing existing `skills/`, `agents/`, and `tools-mcp/` entries.
2. Make setup explicit.
   Declare required secrets and approval gates in `plugin.yaml`.
3. Keep runtime packaging honest.
   The installer will generate `.codex-plugin/plugin.json` or
   `.claude-plugin/plugin.json` on install for those runtimes. Keep the source
   `plugin.json` minimal and runtime-agnostic, and document any manual steps
   that still remain after generation.
4. Treat hooks as risk-bearing.
   Any automatic action should be explained in `README.md`.

## Required manifest sections

- identity: `id`, `name`, `description`, `version`, `released_at`
- placement: `category`, `tags`, `runtimes`
- structure: `entrypoints`
- composition: `includes`
- governance: `requires`, `verification`

## Definition of done

- valid `plugin.yaml`
- clear `README.md`
- at least 5 prompts in `tests/test-prompts.md`
- one example flow in `examples/`
- included component references resolve to real registry entries
- required secrets and approvals are documented
- generated runtime manifests are safe to inspect before enablement
