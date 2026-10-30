# Runtime package compilation

Schema 2.1 self-contained plugin releases are compiled at install time from
`plugin.yaml` and `dependencies.lock.json`. The portable `plugin.json` file is
identity metadata; it is not sufficient evidence that a runtime can discover
or execute the complete package.

The compiler produces a deterministic `.runtime/<runtime>.json` contract for
`codex`, `claude`, or `generic`. The contract registers every locked skill,
agent, MCP server, configuration directory, and declared hook. Each
registration is either `native` or `advisory`; advisory entries include a
reason so unsupported behavior cannot disappear silently.

For Codex, the compiler replaces the source identity stub with an Agent Plugins
1.0 root `plugin.json`. Locked MCP configurations are merged into the portable
root `mcp.json`, including its schema and explicit transport types. Component-
relative commands, working directories, and `PLUGIN_ROOT` references are
rebased to the bundled tool root so merging does not change their execution
context. Remote MCP endpoints are admitted only when their URL, transport
security, and HTTP headers satisfy the Agent Plugins contract; fixed headers
that are credential-bearing or contain credential-shaped values are rejected.
For Claude, the
compiler writes `.claude-plugin/plugin.json` and retains validated nested MCP
configuration paths. Locked skills are
materialized under `skills/`, while Claude agent definitions are compiled into
its native `agents/<name>.md` format. Documentation-only tool specifications
remain visible as advisory entries.

Runtime discovery roots must be clean before compilation. Source archives may
not smuggle additional skills, agents, commands, MCP configurations, native
manifests, or default hook files around the dependency lock. The final
discoverable skill, agent, and MCP server inventories are compared with the
compiled runtime contract.

JSON hook definitions are parsed before they are registered. Native status
requires a supported runtime event, a non-empty handler set, a handler type and
fields implemented by that runtime, and contained references to bundled files.
Malformed hooks fail compilation; runtime-specific or unsupported handler
behavior remains visible as advisory. Codex command and `mcp_tool` handlers are
preserved under their event constraints. Claude command, HTTP, `mcp_tool`,
prompt, and agent handlers use closed type and event matrices; command hooks
preserve and type-check native `args`, `asyncRewake`, and `shell` fields without
widening Codex hook support. Hook filters are checked against their event and
plugin-document scope; fields that Claude ignores remain advisory instead of
being presented as enforced. A document that mixes native and advisory handlers
is rejected with an instruction to split it, preventing advisory quarantine
from silently disabling valid enforcement hooks. Claude HTTP hook credentials
must be runtime environment references declared by `allowedEnvVars`; literal
credential-bearing URL or header content is rejected without being echoed. URL
checks cover decoded path, query, and fragment components, and HTTP is allowed
only for `localhost` or literal loopback addresses; every remote endpoint must
use HTTPS.
Unknown top-level hook-document fields are hard errors rather than advisory
classification, so they cannot move otherwise native handlers out of runtime
discovery.
Markdown or YAML hook guidance is always advisory.

The generic output contract is defined by
`shared/schemas/runtime-package.schema.json`. It deliberately describes
registration and enforcement without pretending that a generic host implements
Codex- or Claude-specific behavior.

Pre-2.1 packages and repository-development installs without an archive-local
`bundled/` closure retain their legacy compatibility path. They do not receive
a `.runtime` contract and are not represented as fully compiled packages.

## Validation invariants

- The dependency lock root must match the plugin identity and version.
- Every locked component appears exactly once in the runtime contract.
- Lock manifest paths use the canonical bundled module layout.
- Native names cannot collide after category prefixes are removed.
- Every registration uses a canonical in-package path that exists.
- Runtime auto-discovery roots contain no entries outside the compiled contract.
- Codex MCP server names and transports match the generated portable
  `mcp.json`.
- Codex MCP launch paths retain each locked component's execution root after
  configurations are merged.
- Remote MCP URLs and headers satisfy the portable transport security rules.
- Native hook documents contain supported events, handlers, and contained
  executable references, with filters effective in that document scope.
- Advisory registrations always explain why runtime enforcement is absent.
- Native manifests and the complete runtime contract are emitted together and
  validated as one compiled package.

Golden artifacts cover Codex, Claude, and generic output. Installer integration
tests verify that a clean runtime receives both the compiled plugin package and
its locked dependency closure without manual file moves.
