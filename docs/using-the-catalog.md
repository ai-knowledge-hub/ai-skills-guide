# Using the Catalog

The catalog separates **what a package is** from **how ready it is to use**.
The normative definitions, evidence rules, target scoping, and expiry semantics
are in the
[Operational Readiness and Promotion Contract](../shared/contracts/operational-readiness-v1.md).

## Catalog projections

The current registry exposes two compatibility projections. They are useful for
discovery, but neither field by itself proves operational availability.

`readiness` describes review maturity and lifecycle:

- `experimental`: not security reviewed
- `reviewed`: a security review is asserted
- `deprecated`: retained for compatibility, no longer preferred

Under the normative contract, `reviewed` maps to `security-reviewed` only when
the exact artifact has a current accepted security disposition. The registry
label alone is not that evidence.

`usability` describes operational behavior:

| Availability | Install result | What remains |
| --- | --- | --- |
| `usable-now` | Instructions or a local executable are copied into the runtime. | Supply task inputs and run the documented first step. |
| `setup-required` | The package is installed, but does not become operational by installation alone. | Configure dependencies, tool bindings, credentials, and policy. |
| `template-only` | A reference contract or scaffold is installed. | Implement and review the missing provider/runtime binding. |
| `documentation-only` | No runtime installation is implied. | Use the material as a guide or contribution map. |

The `execution` value adds the technical shape:

- `instructions`: reusable procedural knowledge for an agent
- `local-tool`: deterministic code runnable on the local machine
- `remote-integration`: client or connector to an external system
- `integration-template`: a provider boundary without a working connection
- `orchestrator`: an agent that coordinates dependencies and requires bindings
- `bundle`: a plugin that installs several catalog entries together
- `documentation`: a learning or architecture pack

## Module behavior

### Skills

Skills are usually `usable-now` instructions. A skill may still depend on local tools or APIs. Installation makes the instructions discoverable; it does not create missing credentials or external services.

### Tools and MCP

Local deterministic tools can be `usable-now`. Remote connectors are `setup-required`. Entries ending in `-template` are normally `template-only` unless their manifest explicitly says otherwise.

### Agents

Agents are orchestrators. Installation copies their specification, but operational use requires every declared skill, agent, and tool dependency plus tool bindings, memory, and governance. Prefer a plugin when one exists because plugin installation resolves bundled dependencies.

### Plugins

Plugins are the installable composition layer. They install declared skills, agents, and tools into native runtime directories and retain hooks, config, templates, and examples inside the plugin directory. Required secrets and approvals still need local configuration.

### Packs

Packs curate learning paths and related catalog entries. They are documentation-only and are not part of the install registry.

## Inspect before installing

```bash
./bin/skills-hub info --module tools --entry adtech/openai-ads-api-client@latest
```

The output includes:

- `usability.availability`
- `usability.execution`
- `usability.requires_setup`
- `usability.limitations`
- `usability.quickstart`
- `usability.source`

An inferred classification is a conservative registry default. A declared
classification has been set in the package manifest and should be preferred
when planning implementation.
Neither `inferred` nor `declared` is verification. A `usable-now` entry is
operationally verified only for the instruction or executable scope and target
key covered by current evidence. Until the registry carries those evidence
references and target keys, inspect the package's documented limitations and
treat the availability value as an unverified classification.
