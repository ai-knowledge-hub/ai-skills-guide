# Authentication and Runtime Readiness

`skills-hub` keeps package installation separate from local authentication.
Released manifests contain authentication methods, required scopes, and
non-secret binding names. Credential payloads remain in a runtime-owned store.

## Configure a binding

Configuration records references to environment-backed credentials; it never
accepts a credential value as a command-line argument or prompt response.

```sh
skills-hub auth configure ads/example-tool@1.0.0 \
  --module tools \
  --method api-key \
  --binding PROVIDER_API_KEY=env:PROVIDER_API_KEY \
  --generation PROVIDER_API_KEY=env:PROVIDER_API_KEY_GENERATION \
  --account account-123 \
  --validator-command provider-auth-check
```

The validation executable must perform a non-destructive provider identity
operation and write one JSON object to standard output:

```json
{
  "authenticated": true,
  "revoked": false,
  "principal": "provider-user-1",
  "account": "account-123",
  "scopes": ["read"],
  "expires_at": "2026-09-17T12:00:00Z",
  "provider": "example-provider",
  "tier": "sandbox",
  "endpoint": "https://api.example.test/v1",
  "region": "global",
  "api_version": "v1",
  "attestation_reference": "provider://identity/check-1",
  "attestation_expires_at": "2026-09-17T12:00:00Z"
}
```

The generation variable is a non-secret revision supplied by the credential
manager and must change whenever the credential is replaced, refreshed,
renewed, revoked, or rebound. The selected credential is supplied to the
validator under the manifest's binding name. Validator commands and arguments
must not contain credentials.
The resolved validator executable is digest-pinned during configuration and is
rejected if its bytes later change. Output is size-bounded, strictly decoded,
and never copied verbatim into an error. Credential-shaped identity metadata is
rejected.

OAuth, device authorization, client credentials, service accounts, workload
identity, brokered authentication, and custom authentication use a packaged
driver at `auth/<method>.json`. The driver is part of the receipt-verified
installation and declares three package-relative executables: `bootstrap`,
`credential`, and `status`, plus the runtimes it supports. Package admission
requires every typed method's driver before either local or remote installation
can mutate runtime state. It validates the canonical driver schema, exact
method/flow identity, executable containment and mode, and compatibility with
every runtime declared by the package. The CLI also rejects a driver that
changes after configuration, reports a different account, or drops a requested
scope.

The bootstrap executable receives a method-specific flow name
(`interactive-browser`, `device-code`, `non-interactive-service`,
`non-interactive-workload`, `brokered`, or `custom`) and the requested account
and scopes as JSON. It owns provider consent and secure credential storage. The
credential executable returns a credential only to the readiness subprocess
boundary, with a non-secret generation; the payload is never written to the
profile. The status executable performs the independent provider observation.
API-key and bearer-token packages may use the same packaged driver convention;
direct environment references remain available for simple runtime-owned stores.

## Inspect readiness

```sh
skills-hub auth status ads/example-tool@1.0.0 --module tools
skills-hub doctor ads/example-tool@1.0.0 --module tools --runtime generic
```

Authentication status distinguishes missing bindings, unavailable validation,
expiry, revocation, wrong account, missing scopes, invalid identity, and ready
state. An environment variable's presence proves configuration only. A current
non-destructive provider observation is required for `authenticated: true`.

`doctor` additionally verifies the installation receipt, installed tree,
runtime, and platform. It does not repair state or run the package operation.

## Run a smoke check

```sh
skills-hub smoke ads/example-tool@1.0.0 --module tools --runtime generic
```

The smoke command verifies the installed tree and authentication immediately
before executing the manifest's declared `execution.smoke_test`. It passes the
same resolved credential snapshot to provider validation and the smoke process.
Child output is discarded because an executable must not be able to copy a
credential into routine logs. The resulting evidence contains artifact and
environment identity, non-secret binding keys, provider identity and scope
observations, validator and execution-runtime fingerprints, timestamps, and a
bounded result reason.

Local-development installations are identified as `local` and do not claim a
published artifact digest. Only a checksum-verified remote installation can
produce smoke evidence bound to a released artifact.

Validator, driver, and smoke subprocesses receive a minimal environment:
required process/runtime variables plus the manifest-declared credential
bindings. They do not inherit `HOME`, the original credential source variable,
or unrelated ambient credentials.

`doctor` persists setup-scoped evidence. `list` and `info` can consume smoke
evidence for an exact installed target with `--runtime`, `--target`, and
`--state-dir`. Before displaying `usable-now`, the CLI revalidates the receipt
and complete plugin closure, artifact and lock digests, executable identity,
environment, provider target, credential generations, scopes, and expiry.
Malformed, stale, substituted, local-only, or mismatched evidence projects as
`not-verified` (or `setup-required` when the live doctor check identifies a
declared setup failure). Smoke and doctor observations are append-only and are
keyed by the complete target identity, so evidence for one runtime or
configuration cannot overwrite another. Catalog projection selects the newest
applicable smoke observation regardless of outcome; an early failure is ordered
using its stable entry, version, module, runtime, platform, environment, and
command envelope even when artifact or authentication fields were not yet
available. A newer failure therefore immediately demotes an older successful
target and re-promotion requires a new qualifying smoke run. Every record is
validated against the canonical smoke-evidence schema before publication and
again before catalog derivation. Evidence publication writes and synchronizes a
temporary file before atomically linking the complete record into its
append-only final path.
