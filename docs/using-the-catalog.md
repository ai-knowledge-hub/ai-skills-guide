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
| `not-verified` | An implementation is present, but installation does not prove it works for your target. | Obtain current target-scoped operational evidence before relying on it. |
| `template-only` | Operational installation is blocked; repository source remains available as a reference. | Copy only as a scaffold, then implement and review the missing provider/runtime binding. |
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

Current skills are `documentation-only` instruction packages. Installation makes their guidance discoverable; it does not authorize executable runtime capability. Seven skills separately disclose packaged local helpers as `not-verified`; installing their instructions does not promote or verify those helpers.

### Tools and MCP

Specification-only connectors and servers are `template-only`. Implemented local tools and clients remain `not-verified` until current evidence supports a target-scoped readiness promotion.

### Agents

Current agents are `template-only` orchestrator definitions. Operational installation is blocked because no launchable orchestrator or verified runtime bindings are packaged.

### Plugins

Current plugins are `template-only` source compositions with advisory hooks. Operational installation, dependency resolution, and runtime-manifest generation are blocked until a self-contained verified runtime bundle is published.

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
- `usability.executable_helpers` and each helper's independent availability
- `usability.source`

All current catalog classifications are declared in package manifests;
repository admission rejects omissions. A declaration is not verification. A `usable-now` entry is
operationally verified only for the instruction or executable scope and target
key covered by current evidence. Until the registry carries those evidence
references and target keys, inspect the package's documented limitations and
treat the availability value as an unverified classification.
