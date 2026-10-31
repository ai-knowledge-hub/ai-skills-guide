# GA4 fixture producer and signed observations

This build continues #84 after #88: it connects a real retained connector to the
scoped evaluator and verifies producer signatures under an independent policy.

## Reviewed release and version inventory

The profile is restricted to `analytics/ga4-mcp-connector@0.2.0`, archive SHA256
`80efc8f9e5176097781da84595a9271b34cc5f1661b098422664cf739e154b84`.
A changed archive is rejected before executing its code. A new release needs
its own reviewed profile; copying an ID or changing a timestamp cannot qualify.

The retained implementation calls Google Analytics Data API `v1beta` directly
through Node's built-in fetch. It contains no provider SDK to version. The
[Google runReport reference](https://developers.google.com/analytics/devguides/reporting/data/v1/rest/v1beta/properties/runReport)
identifies the POST endpoint and request shape used by the host fixture.
Documentation inspection does not establish live provider compatibility.

The artifact supports MCP initialization versions `2024-11-05`, `2025-03-26`
and `2025-06-18`. The check exercises all three handler negotiations; the
emitted
combination records `2025-06-18`, `v1beta`, and the exact measured Node version.
Other combinations do not inherit that record. CI explicitly selects Node 22;
local execution records its own version. No supported-version range is inferred.

## What is checked

A host-owned probe is embedded outside the tested archive. It imports the
validated temporary extraction and checks initialization, tool discovery,
runReport endpoint/method/body, bounded two-page normalization and property
isolation. A negative test changes the endpoint to v2 in a temporary copy and
checks that the host oracle reports incompatibility.

The process receives an empty environment, fixed fixture property configuration,
an injected fixture token and a fixture fetch implementation. Provider
credentials
and Node options from the ambient environment are not forwarded. Checks have a
20-second process deadline. The probe clock is fixed for deterministic sample
report dates; the signed observation time is captured after the real check.

This proves a local API/MCP handler fixture contract on a retained artifact. It
does not verify Google authentication, a live GA4 account, native transport
registration or every provider capability. Evidence uses operation
`ga4-fixture-contract` and provider target `host-owned-local-fixture`.
The connector's own fixture response labels are not promoted to live evidence.

The actual archive bytes are snapshotted, size-limited, digest-checked, safely
extracted and historically validated before inspection. The request binds the
archive digest, actual package version, host probe/configuration digest and
measured dimensions. The dependency-lock identity is SHA256 of its JSON value
(`null` for this release's absent lock), rather than a fabricated lockfile.
The host probe content hash contributes to its test version.

## Signature and authority boundary

Ed25519 signatures cover the exact raw observation bytes, a compatibility-only
protocol domain and the verifier key ID. The envelope contains only schema
version, key ID, base64 observation and base64 signature. It contains no public
key, approval policy or credential.

The verifier receives a separate operator-controlled key policy. Each key is
restricted to the approved producer, package ID, artifact digest, test ID and
version, operation and provider target. Missing policy fields, unknown/revoked
keys, changed bytes, substituted keys and signed scope expansion fail closed.
Successful signature verification only approves bytes from that producer; the
existing evaluator still matches them against an independently selected subject
and requirement set. A valid fixture signature cannot approve a live operation.

The raw manual approval mode remains an offline diagnostic. Signed mode cannot
mix raw approvals and signed inputs. Neither mode is consumed by runtime
admission, and both leave operational readiness unevaluated.

## Running the producer

First inspect the reviewed public scope:

```bash
go run ./cmd/ga4-compatibility-check --describe-profile
```

The description has an empty public key and grants no trust. The operator must
configure the approved public key and key ID independently in a policy JSON
object keyed by key ID. Key distribution and rotation remain operator-owned.
No production signing key or policy is generated or committed by this build.

With an existing operator-owned base64 Ed25519 32-byte seed file:

```bash
go run ./cmd/ga4-compatibility-check \
  --signing-seed-file /private/operator/producer-seed \
  --key-id ga4-fixture-ci \
  --out /tmp/new-ga4-observation
```

The output must be a new directory. It contains request.json, evidence.json and
signed-evidence.json only. The raw evidence bytes exactly match the envelope's
signed bytes. Signing material is kept in the Go host and is not passed to Node,
printed or copied to the observation directory.

```bash
go run ./cmd/compatibility-evaluator \
  --request /tmp/new-ga4-observation/request.json \
  --signed-evidence /tmp/new-ga4-observation/signed-evidence.json \
  --producer-policy /private/operator/approved-producers.json
```

Only an approved key with matching scope can produce tested fixture
compatibility.
Removing that key makes verification fail. A matching fixture observation never
proves live readiness or overrides expiry/revocation in the runtime auth path.

## Verification and remaining integration

Tests run the real retained archive with ephemeral test keys, verify the signed
bytes and policy, evaluate the resulting combination and reject live-operation
reuse. They cover ambient environment isolation, artifact mutation, endpoint
mutation, signature/key/payload substitutions, missing scope fields, cross-
domain
signatures, revoked keys, ambiguous CLI modes and output-directory preservation.

Fixture observations and temporary keys are test data, not published provider
verification. Production key provisioning, independent current-target discovery,
new package declarations/releases, live sandbox observations, catalogue display
and coordinated v2 runtime migration remain under #84/#52. Retained release
bytes and existing runtime credential/provider checks stay unchanged.

## Bounded validation results

- The full Go contract suite, focused race tests, CLI build and manifest/schema
  gates passed locally. Registry generation and release-history checks preserve
  tracked release and index bytes.
- Independent assertions bind the published API/protocol IDs to the reviewed
  contract and compare Node identity with a separate host version query.
- All seven approved scope fields are tested for omission and signed substitution.
  Payload/key/signature substitutions, cross-domain signatures, key-ID rebinding,
  revocation and mixed CLI approval modes are rejected.
- Four selected harmful mutations were detected (4/4): disabling signature
  verification, disabling scope enforcement, forwarding ambient environment and
  removing the host endpoint oracle. This is a finite sample, not whole-system
  mutation coverage or evidence of live provider compatibility.

The independent authority is operator-owned key policy plus the reviewed
artifact/profile and host oracle. Deployment of the production issuer and key
policy distribution remain unimplemented; test-generated keys confer no
production authority. Real live-provider outcomes remain unknown until the
corresponding scoped checks run under #52.
