# Manifest Contract v2

- Status: normative
- Schema version: `2.0`
- Schema definitions:
  [`manifest-contract-v2.schema.json`](../schemas/manifest-contract-v2.schema.json)

## Purpose

Manifest contract v2 makes a package's execution shape, released artifact,
authentication requirements, and verification references machine-readable. It
applies equally to skills, agents, tools, and plugins.

These declarations describe requirements and evidence locations. They do not
by themselves prove installation, authentication, security review, or
operational readiness. Consumers must apply the
[operational readiness contract](operational-readiness-v1.md) to the referenced
artifact, target, and current evidence.

## Versioning and compatibility

`schema_version` is separate from the package's SemVer `version`.

- A manifest without `schema_version` is a legacy v1 manifest. Existing fields
  retain their current meaning and the manifest remains valid.
- A manifest that declares any v2 contract field must set
  `schema_version: "2.0"` and provide `execution`, `artifact`,
  `authentication`, plus verification `evidence` and `last_verified_at`.
- Producers must not emit a partial v2 contract. Consumers that do not support
  schema version 2 must report it as unsupported; they must not reinterpret or
  silently discard its execution or authentication requirements.
- Compatible clarifications may extend the 2.x schema with optional fields.
  Removing fields, weakening invariants, or changing meanings requires a new
  major schema version.
- Registry format `1.1` preserves v2 metadata while continuing to carry legacy
  entries without synthesizing v2 claims.

## Execution

`execution.kind` defines the package's technical shape:

| Kind | Meaning | Required checks |
| --- | --- | --- |
| `instructions` | Guidance without a packaged executable | No command |
| `script` | Packaged interpreter-driven program | Command and smoke test |
| `cli` | Packaged command-line program | Command and smoke test |
| `mcp-server` | Launchable MCP protocol server | Command, healthcheck, and smoke test |
| `service` | Long-running non-MCP service | Command, healthcheck, and smoke test |
| `orchestrator` | Launchable agent or workflow coordinator | Command and smoke test |
| `bundle` | Package that composes other modules | No direct command required |
| `integration-template` | Non-executable integration starting point | No command |

Commands are argument arrays, not shell strings. A consumer must invoke the
declared executable directly and must not add shell interpolation. Platform and
runtime arrays describe supported targets; they do not attest that any target
has been verified.

`instructions`, `bundle`, and `integration-template` must not declare
`command`, `healthcheck`, or `smoke_test`. Scripts, CLIs, and orchestrators must
declare a command and smoke test but no service healthcheck. MCP servers and
services must declare all three executable fields.

## Artifact

Every v2 manifest declares:

- `self_contained`: whether the released artifact contains its claimed local
  runtime closure;
- `dependency_lock`: repository-relative lockfile path, or `null` when no
  dependency manager applies;
- `checksums`: repository-relative checksum manifest path, or `null` when the
  artifact format cannot carry one;
- `sbom`: repository-relative SBOM path, or `null` when no SBOM is yet
  available.

Paths must remain inside the package. `null` is an explicit absence, not proof
that a requirement is unnecessary. Promotion rules may require a non-null
field for a particular delivery or readiness state. Paths use `/` separators;
absolute, drive-qualified, UNC, backslash, mixed-separator, and parent-traversal
paths are invalid on every target platform.

## Authentication

`authentication.status` is `none`, `optional`, or `required`. Supported method
identifiers are:

- `none`
- `api-key`
- `bearer-token`
- `oauth-authorization-code-pkce`
- `oauth-device-flow`
- `oauth-client-credentials`
- `service-account`
- `workload-identity`
- `brokered`
- `custom`

`required` authentication must declare at least one method and at least one
`credential_bindings` name. A binding is a non-secret identifier resolved by
the runtime credential store. Manifests must never contain a secret, token,
private key, authorization code, refresh token, or credential payload.

Schema validation restricts credential fields to binding names. Semantic
manifest validation additionally applies a maintained catalog of
high-confidence provider signatures and secret-assignment forms anywhere in the
manifest, including commands, URLs, scopes, evidence references, and operator
guidance. Diagnostics identify only the field path and never echo the suspected
value. This scan is defense in depth, not proof that arbitrary or newly issued
credential formats are absent; producers remain responsible for preventing all
credential payloads from entering manifests.

Scopes are provider-native strings and must not be broadened by consumers.
`setup_url`, `credential_storage`, `validation`, and `revocation` describe the
operator path. They are guidance, not proof that credentials are present,
valid, authorized, or current.

When status is `none`, the only method is `none`, and bindings and scopes are
empty. Optional authentication may list supported methods without asserting
that any binding currently exists.

## Verification

`verification.evidence` contains opaque references to evidence records.
`verification.last_verified_at` records the latest verification observation
represented by those references. Neither field may be self-issued as an
operational claim.

Consumers must resolve evidence through its authoritative store and check its
producer, scope, target key, artifact digest, contract version, observations,
freshness, and disposition. A timestamp without qualifying evidence does not
promote readiness. Credential rotation, target changes, artifact changes, and
expiry invalidate evidence as defined by the operational readiness contract.

## Responsibility and authority

| Claim | Manifest author may declare | Deterministic validator proves | External/runtime evidence proves |
| --- | --- | --- | --- |
| Execution | Intended kind, command, checks, targets | Structural completeness and safe representation | Entrypoint launches and behaves correctly |
| Artifact | Expected lock, checksum, and SBOM paths | Paths are well formed | Published artifact contains and matches them |
| Authentication | Supported methods, bindings, scopes, operator guidance | Typed method and binding invariants | Current principal, scopes, provider target, and credential generation |
| Verification | Evidence references and observation timestamp | Required fields and timestamp shape | Evidence validity, freshness, outcome, and accepted disposition |

The model or catalog may explain these declarations. It may not manufacture
provider identity, credential validity, evidence, approval, or readiness from
them.

## Consumer behavior

Registry builders must preserve all v2 fields without deriving missing values.
Installers and launchers must fail closed on unsupported execution kinds,
platforms, runtimes, or authentication methods. Unknown future schema versions
must remain visible as unsupported rather than being treated as legacy.

Repository admission additionally verifies that declared entrypoint, artifact,
and package-relative command paths exist and remain contained in the package.
It resolves internal skill, agent, tool, and plugin-include references against
the repository, rejects incompatible runtime closure and dependency cycles, and
requires a current verification reference and timestamp for an explicit v2
`usable-now` claim. These checks do not turn an opaque evidence reference into
proof: release promotion must still resolve the authoritative evidence record
and validate its target, artifact digest, observations, producer, disposition,
and expiry under the operational readiness contract.

Legacy v1 manifests receive structural entrypoint and dependency checks but are
not promoted on the strength of legacy verification labels. Their remaining
inferred usability classifications are a compatibility projection pending
explicit catalog reclassification.

The golden fixtures under `shared/schemas/fixtures/manifest-v2/` cover every
execution kind and authentication method, all four module schemas, explicit
artifact absence, required binding enforcement, executable smoke-test
enforcement, and rejection of embedded secret-shaped fields.
