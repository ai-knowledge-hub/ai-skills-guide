# Native Codex discovery acceptance — 13 November 2026

<!-- markdownlint-disable MD013 -->

## Result and exact subject

**Passed on macOS ARM64:** the actual released installer
`v0.4.0-alpha.1` installed the published
`engineering/implementation-strategy@0.1.0` package, and official Codex CLI
`0.168.0` reported the expected native discovery outcomes. The
[JSON report](native-codex-2026-11-13/released-macos.json) retains the exact
release, digest, initialized-home and probe bindings. The
[method and reproduction guide](../native-codex-discovery.md) explains the
authority model and finite scope.

| Case | Delivery | Native discovery / controls |
| --- | --- | --- |
| Default user target, no `CODEX_HOME` or `--target` | Exact bytes and receipt verified under `~/.codex/skills` | All five states passed; expected name/path, enabled `user` scope |
| Explicit `~/.agents/skills` target | Exact bytes and receipt verified | All five states passed; expected name/path, enabled `user` scope |
| Explicit repository `.agents/skills` target | Exact bytes and receipt verified | All five states passed; expected name/path, enabled `repo` scope |
| Existing disposable `CODEX_HOME`, no `--target` | Exact bytes and receipt verified under its `skills` directory | All five states passed; expected name/path, enabled `user` scope |
| Arbitrary target outside discovery roots | Exact bytes and receipt verified | Absent both before and after installation |

The four positive cases followed absent → installed → missing instructions →
malformed instructions → restored published bytes. All **22/22 native discovery
probes** passed. Each case reused its app-server process with forced reloads.
Malformed controls produced a diagnostic for the exact instruction path. Every
scan left the installed package unchanged; restored packages passed byte/receipt
verification again. Five disposable homes/repositories were cleaned up.

The default-path concern was not reproduced with this client. Both the existing
legacy default and explicit documented `.agents` locations work, including the
two-level `engineering/implementation-strategy` namespace. No installer resolver
change or new CLI release is needed for this measured combination.

## Identity and provenance

| Subject | Identity |
| --- | --- |
| Installer tag | `v0.4.0-alpha.1`; GitHub release checksum and repository/workflow/tag-bound signed provenance verified before execution |
| Installer SHA-256 | `1e26906d6f2e7fba2b3807953683a47f9caa5b6e59db15c41bc5f3ec08f9f34e` |
| Codex release | `rust-v0.168.0`, `codex-aarch64-apple-darwin.tar.gz` |
| Codex archive SHA-256 | `c94dffc075ce5527a55d060f35412b45d1c1c1d16306a337dcf0d8e3ecfebc98` |
| Codex binary SHA-256 | `1b7ee0f02650f3e2d31f50232699f667a1cbe60f431a57e118f2e785fe372c4b` |
| Package archive SHA-256 | `6e0946ab1e76fc8769171fa874f8cd4373e47655559a26c90cca4914efe5ab72` |
| Manifest SHA-256 | `79bbfdb791b739c6fa8396ba27fb3de48354707f00932221f0e4b1f55e2ab199` |

Codex archive digests are reviewed static pins from the official release's asset
metadata, not a separate client build attestation. The native client was
extracted into temporary space, without replacing the installed application.
The read-only Go helper is a checker dependency, not a candidate installer
substituted for the published binary. Its digest is retained in the JSON report.

## Verification

- Native released-installer run: five isolated cases, 22 probes, all passed.
- New deterministic checker suite: **12 tests passed**, including archive and
  record substitutions, five target environments, lifecycle orchestration,
  package mutation and nine protocol fault cases.
- Existing published-install/MCP/plugin/Windows boundary suites:
  **70 tests run, 69 passed and one Windows-only case skipped** on macOS.
- `go test ./...` passed.
- Required structure, registry generation and manifest/schema checks passed;
  generated registries, retained releases and readiness contracts are unchanged.
- Both touched workflows passed `actionlint` v1.7.7, with optional shellcheck and
  pyflakes integrations disabled. `git diff --check` passed.
- All four changed Markdown files passed `markdownlint-cli2` v0.19.0.

Initial Go and existing Python runs in the restricted tool sandbox could not
bind their loopback fixtures. The same suites passed when rerun with loopback
available. These were environment failures, not changed application behavior.

The new manual workflow enables separate Ubuntu and macOS reports; it has not
been dispatched from this local feature branch. Public network checks remain
separate from deterministic PR checks. No web source changed, so browser/layout
checks were not repeated for this verification-only build.

## Limits and follow-up

This proves client-reported discovery for one published skill, released installer,
client version and measured host. It does not prove skill invocation or adherence,
desktop UI behavior, model output quality, Linux execution, other versions,
Claude adoption, plugin/MCP registration, provider access or live readiness.
The test passes no credentials and starts no model turn. It does not claim
OS-wide egress monitoring or a fully sandboxed native process.

Issue #52 and epic #33 stay open. Next: obtain the independent Linux native
discovery report, then add the separate real Claude client adoption case.
