# Runtime model readiness attestation

This contract is preserved for the **development-only readiness profile**.
It is not part of the first installer release. A successful package installation
does not verify model capabilities or require a model attestation. See
[installer release scope](cli-installation-release.md).

The development preflight fails closed until the selected runtime supplies a
signed, short-lived model capability attestation. The runtime adapter owns the
private key; the development binary pins the corresponding Ed25519 public key.
Neither the install caller nor the installed package can replace that authority.

`make cli-build-runtime-dev` requires explicit
`CODEX_MODEL_ATTESTATION_PUBLIC_KEY`, `CLAUDE_MODEL_ATTESTATION_PUBLIC_KEY`, and
`GENERIC_MODEL_ATTESTATION_PUBLIC_KEY`, and produces
`bin/skills-hub-runtime-dev` with the `runtime_readiness` build tag.
The normal `make cli-build`, candidate build and release builder do not read or
embed these roots. A release build uses `installer_release`; combining that tag
with `runtime_readiness` is a compile error. There is no runtime flag to enable
these development commands in the released installer.

`make cli-runtime-test` retains the signed-attestation integration test with an
isolated generated keypair. Private production keys remain runtime-adapter
secrets outside this repository. A direct development build without roots still
fails closed; it cannot manufacture model authority.

The adapter writes a JSON document with this closed contract:

```json
{
  "schema_version": "skills-hub.model-attestation/v1",
  "runtime": "codex",
  "agent_id": "marketing/weekly-performance-supervisor",
  "agent_version": "0.2.0",
  "runtime_contract_sha256": "<digest from the install receipt>",
  "compiled_contract_sha256": "<SHA-256 of .runtime/codex.json>",
  "model_identity": "<provider/model identity>",
  "model_version": "<provider model version>",
  "capabilities": ["tool-use", "structured-output"],
  "issued_at": "2026-09-26T12:00:00Z",
  "expires_at": "2026-09-26T13:00:00Z",
  "signature": "<standard-base64 Ed25519 signature>"
}
```

The signature covers the compact JSON encoding of all preceding fields in the
displayed order, excluding `signature`. Validity may not exceed 24 hours. The
runtime, agent identity/version, both contract digests, model identity/version,
capabilities, and timestamps are therefore authenticated together.

Pass the adapter-produced document to preflight:

```bash
./bin/skills-hub-runtime-dev run-agent \
  --agent marketing/weekly-performance-supervisor \
  --runtime codex \
  --model-attestation "$MODEL_ATTESTATION_PATH"
```

`run-agent` first selects the trust root from its compiled development runtime
identity, then
records the verified model identity, version, capabilities, and the
attestation SHA-256 in its audit report. Missing, expired, replayed, malformed,
or under-capable evidence blocks readiness. The command does not accept a
public-key override, install-time trust input, environment trust input, or
caller-authored capability flags. A distributor that changes a runtime trust
root must authenticate a new readiness-enabled build; it cannot rotate authority
through an agent installation.

When roots are omitted, `run-agent` uses the same Codex or Claude module roots
as installation. Generic runtimes have no implicit installation root, so all
three root overrides remain required.
