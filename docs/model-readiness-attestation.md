# Runtime model readiness attestation

Installable agents fail closed until their selected runtime supplies a signed,
short-lived model capability attestation. The runtime adapter owns the private
key. The authenticated `skills-hub` release pins the corresponding Ed25519
public key as a release-governed build input for each runtime identity. Neither
the install caller nor the installed agent package can select or replace it:

```bash
./bin/skills-hub install \
  --module agents \
  --entry marketing/weekly-performance-supervisor@latest \
  --runtime codex
```

Governed builds require `CODEX_MODEL_ATTESTATION_PUBLIC_KEY`,
`CLAUDE_MODEL_ATTESTATION_PUBLIC_KEY`, and
`GENERIC_MODEL_ATTESTATION_PUBLIC_KEY`. `make cli-build`, CI, and the release-tag
workflow inject those repository-managed public roots into the matching
`internal/agents` linker variables. A missing root fails the governed build
instead of producing an unusable release. Direct development builds contain no
authority and fail closed. CI also builds a real CLI with an isolated test-only
root and proves that a matching signed attestation reaches `ready`. The private
keys are runtime-adapter deployment secrets and must never enter this repository.
Users must verify the CLI through its normal distribution provenance; a locally
modified binary is a different trust domain.

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
./bin/skills-hub run-agent \
  --agent marketing/weekly-performance-supervisor \
  --runtime codex \
  --model-attestation "$MODEL_ATTESTATION_PATH"
```

`run-agent` first selects the trust root from its compiled runtime identity, then
records the verified model identity, version, capabilities, and the
attestation SHA-256 in its audit report. Missing, expired, replayed, malformed,
or under-capable evidence blocks readiness. The command does not accept a
public-key override, install-time trust input, environment trust input, or
caller-authored capability flags. A distributor that changes a runtime trust
root must publish and authenticate a new CLI build; it cannot rotate authority
through an agent installation.

When roots are omitted, `run-agent` uses the same Codex or Claude module roots
as installation. Generic runtimes have no implicit installation root, so all
three root overrides remain required.
