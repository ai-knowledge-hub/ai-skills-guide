# Compatibility policy implementation

The target policy is
[operational readiness v2](../shared/contracts/operational-readiness-v2.md).
Tracking: #83 policy; #84 compatibility implementation; #85 tests; #86 clocks.

## Implemented foundation

- Manifest schema 2.2 permits optional compatibility declarations for API, SDK,
  protocol, runtime and platform requirements. Earlier schemas reject the field.
- Each requirement names exact versions and/or capabilities. Ranges are deferred
  until a comparison scheme and support authority are defined. Provider date
  versions and opaque identifiers are preserved rather than treated as semver.
- Parser, canonical registry projection and runtime contract digests preserve
  and bind the declarations. Older manifests remain unchanged.
- CLI info labels declarations explicitly. Historical selections do not inherit
  the latest entry's declarations. All four web detail families show declared
  requirements or explain that compatibility is unknown.
- Declarations cannot supply a tested-compatible status or evidence result.
  Compatibility metadata does not alter authentication or admission behavior.

Example for a future release (illustrative requirements, not a test result):

```yaml
schema_version: "2.2"
compatibility:
  - kind: api
    name: Example Data API
    versions: [v1beta]
  - kind: sdk
    name: example-client
    versions: ["3.2.1"]
  - kind: protocol
    name: Rolling API
    capabilities: [bounded-query]
```

Quote numeric-looking version identifiers in YAML to keep strings.
No existing package has been relabelled or released with these example values.
Registry 1.3 remains the transport for additive declarations; older manifest
consumers reject schema 2.2 on operational admission. This is not a v2 runtime
readiness rollout, and existing v1 evidence gates remain active.

## Remaining work under #84

1. Inventory actual API/SDK versions from connector code, dependency locks and
   provider contracts. Publish new releases with truthful declarations; preserve
   retained archives and manifest identities.
2. Define independent test observation records scoped to artifact, operation,
   runtime, provider and dependency versions. Implement exact combination
   matching, unknown/incompatible results and change-based invalidation.
3. Derive composed compatibility from each actual required/optional dependency;
   expose tested versus declared coverage in the CLI and catalogue.
4. Migrate runtime/release gates together under the v2 policy. Replace the
blanket
   manifest-age gate only after compatibility evaluation and current credential,
   target and effect-authority checks are integrated and tested.
5. Exercise older/newer producer-consumer and rollback combinations, provider
   capability changes and genuine credential expiry/revocation. Reconcile #52's
   release criteria before declaring the migration complete.

## Validation of this foundation

- `make cli-test`: passed, including schema/version rejection, declaration
  preservation, historical CLI selection and runtime-contract binding tests.
- `make registry` and `bash scripts/validate-manifests.sh`: passed; canonical
  indexes and retained release files have no diff.
- `pnpm build`: passed, including lint/type checks and 76 static pages.
- `PLAYWRIGHT_BROWSER_CHANNEL=chrome pnpm test:e2e`: 16 passed, including unknown
  compatibility across all four detail families. Chrome is the established local
  fallback; CI retains its managed browser.
- `git diff --check` and release-history checks: passed.
