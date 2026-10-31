# Catalogue and manifest test assurance

Reviewed 31 October 2026 for #85 with the private failure-space-test-design
skill. This review covers static catalogue integrity, declaration projection,
release-bound editorial notes and their display. It does not establish live
provider compatibility or settle the runtime age policy in #83.

## Evidence and domain model

The skill's discovery scan inventoried 1,084 files, including 167 test-related
files. These are discovery counts, not coverage or required catalogue totals.
The following evidence was then inspected directly.

| Authority / boundary | Confirmed behavior and evidence |
| --- | --- |
| Maintainer -> source manifest -> canonical index | [Manifest contract v2](../shared/contracts/manifest-contract-v2.md) requires explicit usability, structural validation and resolved dependencies. `internal/registry/builder.go` preserves declarations; time-sensitive projections are separate. |
| Release publisher -> retained bytes -> installer | [Release store](../releases/README.md), archive/receipt tests and `scripts/check-release-history.sh` protect release identity and immutability. A catalogue label is not installation authority. |
| Editor -> exact release note -> website | [Catalogue contract v1](../shared/contracts/catalogue-compatibility-v1.md) requires exact module/ID/version matching, provenance, unknown fallback and informational review dates. |
| Plugin -> included provider -> effect claim | The advertising lab's `plugin.yaml` and [README](../plugins/marketing/chatgpt-advertising-experiment-plugin/README.md) specify its OpenAI Ads read client, offline mock path and unavailable writes. The named product requirement survives a coordinated metadata deletion. |
| Runtime user -> credential observation -> current status | Operational v1 and `internal/readiness/readiness_test.go` distinguish expiry, revocation, wrong account/scope and changed credential generation. Static annotations do not grant authority. |

Actors are maintainers, release publishers and catalogue/CLI users. Material
failures include lost package identities, substituted providers, hidden helpers,
expanded effect claims and notes attached to the wrong release. Source manifests
and editorial notes are reviewed inputs; canonical indexes and rendered panels
are derived views. Publication and installation are separate commit points.

**Confirmed:** the website has no consumer of the retired compatibility
evaluator. It displays maintainer information. **Unknown:** code review and
fixture success do not establish current vendor support or account access.
**Historical conflict:** earlier #85 wording asked for tested-compatible and
known-incompatible evaluator states. The accepted static-site contract supersedes
that scope; these tests do not recreate an evaluator or provider service.

## Independent requirements and test portfolio

The classification regression anchors are a required subset, not a census:
the seven named helper skills in `skillHelpers`, seven current orchestrators,
eleven current plugins and eight current tools. Their identities and useful
role distinctions are deliberate requirements, separate from the builder's
output. New valid entries need no unrelated total or membership update. Intentional
retirement or changed product capability requires reviewing the affected anchor.

The [tool review inventory](tool-version-review.md) independently identifies the
seven reviewed releases. Additional releases need not inherit a note or acquire
one automatically. Historical notes may remain when a new release appears.

| Invariant / consequence | Oracle and executable evidence |
| --- | --- |
| Valid growth preserves existing release identity and declarations | `TestCatalogueAdditionPreservesExistingEntries` builds isolated repositories for all four modules; exact pre-existing entries remain equal after addition. |
| Required identities cannot disappear behind equal counts | `requireCatalogueIdentities` uses explicit named requirements. Its six-case table accepts addition/reordering and rejects deletion, valid substitution and duplication. |
| A required dependency cannot vanish with its incoming edge | `TestCatalogueWorkflowRequiresNamedDependencyDespiteCoordinatedDeletion` exercises actual repository validation and indexing. A node+edge deletion is structurally self-consistent but still fails the independently named workflow requirement. |
| Existing declarations and separate helper classifications remain truthful | `TestCatalogueDeclarationsMatchRetainedRelease` compares all manifest-derived fields with immutable release records, including instruction-only skills outside the named anchors. `TestCatalogClassificationsAreDeclaredAndTruthful` requires each named helper and its not-verified/local-tool label, permits additional helpers, and requires declared usability/limitations on all entries. |
| Provider identity and effect scope cannot be rewritten together | `TestAdvertisingLabPreservesRequiredProviderAndEffectBoundary` requires the OpenAI Ads client on both include and required/read-only authority paths and explicit unavailable writes. Six coordinated/unilateral mutations are rejected. Existing admission authority checks remain. |
| Schemas validate behavior, and declarations cannot self-issue proof | `TestManifestSchemasRejectExecutableHelpersForV20` runs the real validator across four modules and three schema versions, with a valid control in every case. Existing canonical-definition drift checks remain. `compatibility_test.go` uses literal expected declarations, missing-metadata checks, old-reader rejection and invalid range/kind/self-issued-status fixtures. |
| Editorial content binds to retained releases and preserves declarations | `catalogue-contract.spec.ts` checks unique retained subjects, required reviewed identities, source/date shape and exact tuple matching. Registry decoration is compared with raw committed indexes, preserving usability, authentication and every other field. |
| Display remains truthful and dates do not promote readiness | The real shared panel is rendered for unknown, native declared and editorial reviewed states. The reviewed fixture is dated 2000-01-01; it remains visible with limits/source and the no-live-check label. Every state retains Not verified. Existing real-page smoke tests check all reviewed tool pages. |

This ledger defines eight scoped invariants. Membership and dependency checks
share identity authority; those interactions are tested with addition, deletion,
same-count substitution and coordinated node+edge deletion. Provider membership,
access and effect claims form a second cluster, tested with coordinated provider
substitution, authority expansion and write promotion. Release identity, source
and readiness disclosure form the third cluster, tested at lookup, decoration
and actual rendering boundaries. Arbitrary unrelated Cartesian combinations and
live network faults are outside this static failure model.

## Counter-review and bounded measurements

- Builder growth: 4/4 module cases.
- Membership partitions: 6/6 selected cases, including three valid cases and
  three harmful identity cases.
- Workflow closure: 5/5 selected cases, including three harmful dependency
  mutations; coordinated node+edge deletion is checked by the independent oracle.
- Advertising-lab relationships: 6/6 selected harmful input mutations rejected.
- Helper schema boundary: 12/12 module/version cases, each with a valid control.
- Exact annotation lookup: 252/252 current module/ID/version combinations. The
  matrix expands with new data; no fixed catalogue size is asserted.
- Display: 3/3 metadata states, including an old editorial review and precedence
  of a matching annotation over a native declaration.
- Producer mutation check: 6/6 selected harmful code mutations detected; no
  survivors in this selected set. Mutations substitute BigQuery's identity,
  hide the reporting helper, promote the reconciler, or ignore module, ID or
  version during annotation selection.

Run `python3 scripts/test-catalogue-mutations.py` with the normal Go, Node 20,
pnpm and Playwright environment. Local Chrome can be selected with
`PLAYWRIGHT_BROWSER_CHANNEL=chrome`. The script first requires passing controls,
then applies one exact mutation at a time and restores the original bytes in
`finally`. Run it serially while no other process edits the two mutation targets.
Infrastructure failures and missing mutation sites are not counted as detections.
It is an optional counter-review tool, not a new CI service or a provider probe.

## Retained checks, cost and limits

The full Go suite retains schema fixtures, archive/closure installation,
digest/receipt binding and runtime authentication tests. In particular,
`TestStatusDistinguishesAuthenticationStates` covers ready, expired, revoked,
wrong-account and wrong-scope observations at a supplied instant;
`TestStatusRejectsGenerationChangeDuringValidation` covers credential replacement.
Composite authority/expiry tests and the v1 before/at/after evidence deadlines
remain unchanged. A supported API declaration cannot override those checks.

The removed assertions were whole-production totals, blanket module-wide
classifications and schema-version pins, a ban on additional helpers and a test
of the schema's particular `allOf/if/then` arrangement. Named product requirements
and the independent immutable-release oracle preserve existing classification
checks while permitting new package roles and schema upgrades. Canonical schema
drift checks and behavioral validation retain their distinct purposes. Literal
counts in small purpose-built fixtures remain useful and were not removed
indiscriminately.

The focused registry suite runs in roughly one second locally. The added web
data checks take milliseconds and the shared-panel test needs one local browser
page. Full browser smoke and release/installer checks retain their existing cost;
no exhaustive per-SDK matrix, provider calls or sleeps were added.

Residual gaps: these tests cannot prove the semantic correctness of every new
package's prose, every provider contract, or every unanchored package's product
role. New material requirements need independent anchors. HTTPS/source shape
does not prove provenance content is authoritative. The existing mutable
package-evidence test clock and remaining calendar-based operational fixtures
are #86; the justification/migration of the 30-day runtime gate remains #83.
Neither was changed by this test refactor.
