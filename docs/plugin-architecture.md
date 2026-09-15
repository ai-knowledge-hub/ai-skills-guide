# Plugin Architecture

## Purpose

Plugins are the packaging layer of the hub. They bundle reusable skills,
agent templates, tool references, hook definitions, and setup guidance into
one installable unit.

## Position in the stack

- `skills/`: task-level expertise
- `agents/`: orchestrated behavior templates
- `tools-mcp/`: integration connectors and MCP definitions
- `plugins/`: portable bundles that compose the layers above

## Plugin package shape

```text
plugins/
  <domain>/<slug>/
    README.md
    plugin.yaml
    plugin.json
    examples/
    tests/
      test-prompts.md
```

## Registry model

The registry index exposes plugin metadata plus composition fields:

- `includes.skills`
- `includes.agents`
- `includes.tools`
- `includes.hooks`
- `requires.secrets`
- `requires.approvals`

## Current first-wave plugin categories

- `marketing-plugins/reporting`
- `marketing-plugins/audit`
- `marketing-plugins/seo`
- `marketing-plugins/content`
- `marketing-plugins/intelligence`
- `marketing-plugins/creative`

## Runtime install behavior

Plugin installs preserve the package directory, resolve referenced
dependencies into their native runtime directories, and, for supported
runtimes, also generate a runtime-specific manifest inside the installed
plugin folder:

- `codex` -> `.codex-plugin/plugin.json`
- `claude` -> `.claude-plugin/plugin.json`

Dependency behavior:

- `includes.skills` -> runtime `skills/`
- `includes.agents` -> runtime `agents/`
- `includes.tools` -> runtime `tools-mcp/`
- `includes.hooks` -> packaged under the plugin's own `hooks/`

Published self-contained plugins carry their complete local closure under
`bundled/skills/`, `bundled/agents/`, and `bundled/tools-mcp/`. Archive
admission validates every declared embedded package and rejects missing or
undeclared bundled package roots. The immutable archive digest therefore binds
the plugin to the exact dependency content it ships. Remote installation also
applies operational admission to every bundled manifest and activates those
packages in the sibling runtime module directories. The dependency trees and
plugin are committed as one recoverable transaction, with the plugin activated
last. In-place upgrades of an installed multi-package closure are rejected:
runtime readers do not participate in installer locks, so sequential directory
replacement cannot provide safe mixed-version visibility. Supporting that path
requires a future versioned closure with one runtime-wide activation pointer.

These generated files are derived from the package `plugin.json` and stamped
with the target runtime. They are scaffolding artifacts, not proof that the
plugin is production approved.

## Current boundaries

This implementation is still registry-first. The installer prepares runtime-
specific plugin manifests for `codex` and `claude`, but teams are still
responsible for reviewing bundled components, wiring secrets, and enforcing
approval rules before enabling plugins in live environments.
