# Using the Catalog

The catalog separates **what a package is** from **how ready it is to use**.
The normative definitions, evidence rules, target scoping, and expiry semantics
are in the
[Operational Readiness and Promotion Contract](../shared/contracts/operational-readiness-v3.md).

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
| `usable-now` | Inspect the operation and target covered by the recorded evidence. | Confirm that evidence matches the intended environment. Static publication does not promote retained declarations to this state. |
| `setup-required` | The entry identifies required configuration, credentials or runtime bindings. | Follow the setup requirements and verify operation for the intended target. |
| `not-verified` | Check the entry's implementation and installation information. This classification alone does not establish installability. | Obtain current target-scoped operational evidence before relying on it. |
| `template-only` | Operational installation is blocked; repository source remains available as a reference. | Copy only as a scaffold, then implement and review the missing provider/runtime binding. |
| `documentation-only` | No runtime installation is implied. | Use the material as a guide or contribution map. |

The `execution` value describes resource form. The site presents these values
in plain language separately from their availability badge:

- `instructions`: agent instructions
- `local-tool`: a local program
- `remote-integration`: an API or MCP integration
- `integration-template`: a provider boundary without a working connection
- `orchestrator`: an agent workflow with coordination and binding requirements
- `bundle`: a component bundle, whose own classification determines whether
  it is a template or an installable package
- `documentation`: a guide

## From inspection to operation

Related reading is optional. Selected resources link to a Publication article
or Harness learning guide with a specific reason to read it. The editorial
mapping in `apps/web/lib/publicationLinks.ts` supports skills, agents, plugins
and tools, records review provenance, and pins release-specific examples to
their exact resource version. Unmapped resources have no related-reading panel.
These links do not change resource readiness, compatibility or installation
requirements, and the linked implementation may have a different scope.

Each resource detail starts with purpose, resource form, inputs, outputs,
prerequisites and a suitable first action. Summaries use current metadata and
retained release files. Editorial explanations identify an exact module,
package and version, with their source filenames beside the release link.
Missing information is shown explicitly.

Bounded previews may omit large, empty or non-UTF-8 files. Detail pages identify
unavailable previews while retaining catalogue metadata, remaining contents and
release links. Catalogue metadata is attributed separately from previewed files.

Instruction packages lead with reading and copying. Applying pasted
instructions depends on the receiving agent app or chat's supported behaviour.
Templates lead with inspecting the reference. Integrations lead with reviewing
setup requirements and access limits. Documented offline samples identify the
runtime, installed-package working directory, sample input and expected result.
Sample fixtures and expected outputs are synthetic examples, distinct from
current execution or provider evidence.

File-installation commands and technical metadata are separate native
disclosures. Requirements, compatibility notes, helper status and complete
instructions remain readable without JavaScript. Copy controls require
JavaScript and clipboard access; selectable text remains available to read.

Detail pages separate the following steps:

1. **File installation** places the package files and generated configuration in
   selected directories. Instruction files can also be read or copied directly.
2. **Native discovery** confirms that the agent application finds and accepts
   those instructions, tools or plugin registrations.
3. **Mock execution** exercises sample fixtures. It establishes only the
   observed sample behaviour, without proving connected provider operation.
4. **Connected operation** requires current configuration, provider
   authentication, scoped permissions and a successful first operation in the
   intended environment. Human approvals remain specific to the intended effect.

File delivery alone does not meet the normative contract's full `installed`
predicate, which also requires complete dependency integrity and accepted
runtime registration. The website's commands explain file installation, while
runtime checks determine native discovery and operation. A mock result cannot
establish live credentials, provider access or permission to perform writes.

The installation section explains how to obtain the released installer before
showing commands. Choose a declared Codex, Claude or generic adapter to open its
exact-version HTTPS installation command. Native disclosures keep each option
readable without JavaScript. Generic commands include an explicit target; native
adapter defaults and environment overrides are described beside the command.
The full retained closure determines executable environments and the intersection
of supported installer hosts. Agent and plugin dependencies can write other
module directories, shown beside the target. Installer binary availability is
separate from package host support: for example, the weekly agent's bundled MCP
servers exclude Windows. All selected environments must already be available;
their declarations do not install or attest runtimes. If bundled requirements
cannot be read, no command is generated from incomplete information.
Use the reported installed package directory for the documented first-use step,
and `skills-hub help setup` for the public installer's offline setup guide.
See [getting started](getting-started.md#install-published-files) for acquisition
and the separately labelled developer-checkout alternative.

Review maturity, security declarations and lifecycle are displayed separately.
A security declaration or a compatibility note does not verify operation.

## Module behavior

### Skills

Instruction-primary skills supply readable procedures. Their instruction files
can be installed or copied for an agent to discover. Skills that include local
programs disclose those helpers independently, with their own setup, limits and
evidence. Reading or copying the guidance does not verify a helper's execution.

### Tools and MCP

Specification-only connectors and servers are `template-only`. Implemented local tools and clients remain `not-verified` until current evidence supports a target-scoped readiness promotion.

### Agents

Agent packages define coordinated work, expected outputs, memory, governance
and runtime bindings. Check each detail page for installation requirements,
provider setup and missing capabilities. For example, the campaign QA agent 0.2.0
requires a campaign configuration reader that is currently unavailable.
Generated configuration still needs native discovery and environment-specific
verification before connected use.

### Plugins

The catalogue includes both source-composition templates and implemented
bundles. A template can be inspected and adapted, and its detail page offers no
operational installation command. Eligible bundles provide file-installation
commands and expose their locked components and hooks. Installability does not
establish native discovery, authentication or operation in a particular target.

The ChatGPT Advertising Experiment Plugin 0.2.0 has distinct mock-lab,
live-read and live-write declarations. Its live writes are **unavailable**
because no supported OpenAI Ads mutation adapter is included. Control-plane,
executor or approval configuration cannot supply that missing implementation.
The UI links this explanation to the exact retained release while preserving
its original `not-verified` field. A newer release requires its own review.

The OpenAI Ads API Client's reviewed release notes independently disclose
missing automatic pagination, retries and rate-limit backoff. Related guides or
sample workflows cannot supply those features to the installed client.

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
