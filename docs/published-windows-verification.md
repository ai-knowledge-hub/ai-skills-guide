# Published Windows instruction installation

<!-- markdownlint-disable MD013 -->

This focused #52 slice runs the actual published `v0.4.0-alpha.1` Windows AMD64
CLI against `engineering/implementation-strategy@0.1.0`. It extends published
installation evidence to Windows without changing the released CLI, immutable
package archives or catalogue readiness.

## Authority and boundaries

| Claim | Independent authority and observable check |
| --- | --- |
| Correct client | Exact `skills-hub_v0.4.0-alpha.1_windows_amd64.exe` asset, checksum and signed repository/workflow/tag-bound provenance, checked before execution; no candidate fallback |
| Correct package | Independently downloaded HTTPS registry, exact-release archive and digest-bound historical manifest; complete installed payload bytes and receipt identity/artifact/contract/tree binding |
| Isolated adoption | Disposable working directory, target, cache and Windows process temp outside the checkout; native argv paths deliberately include spaces |
| Safe duplicate rejection | Nonzero exit and unchanged `lstat` inventory, bytes, file types, links, modes/read-only state, attributes, file IDs, link counts and exposed timestamps |
| Cache recovery | Damage the disposable `SKILL.md`, then require offline forced reinstall to restore the published bytes and a valid receipt |
| No directory redirection | Reject symlinks, Windows junctions and all other reparse points before traversing an installed package; snapshots retain redirect metadata without following it |

The read-only Go verifier is built on the runner with `CGO_ENABLED=0`. It is a
checker dependency, not the installing client. The CLI and verifier are copied
into the disposable workspace with their `.exe` suffixes intact. Downloaded
release binaries are invoked only after successful checksum and provenance
verification. Windows does not provide POSIX execute-permission gating via
`chmod`; the execution ordering is the trust boundary.

Children receive `SYSTEMROOT` and a PATH limited to its `System32` directory.
Installer children also receive `TEMP` and `TMP` pointing to a new directory
inside that case's disposable workspace. No inherited user profile, application
data directory, runtime home, provider credential, Node option, proxy setting or
GitHub token is supplied to them. The trusted parent uses its read-only CI token
only for GitHub metadata/provenance verification under the existing origin rules.

The offline transition retains the existing non-listening loopback HTTP(S) proxy
guard. This is not an OS-wide network sandbox. NTFS metadata checks do not claim
a complete ACL/security-descriptor audit. Normal file inventory and bytes,
receipt integrity and safe duplicate behavior remain required on all hosts;
POSIX permission checks are retained on Linux/macOS.

## Run and acceptance

Dispatch **Published registry installation** with:

- `sample: instructions`
- `client_source: release`
- `cli_release: v0.4.0-alpha.1`
- blank `registry_url` to use the public skills index.

This runs the existing Linux/macOS jobs plus `windows-instructions` on the
Windows hosted runner. Other samples and candidate-mode dispatches skip that
Windows job. The checker independently rejects Windows MCP/plugin samples before
downloading or executing a client, including when invoked outside the workflow.

Inspect `published-install-windows-latest-release/report.json`. Require a passed
released-client report on Windows AMD64, the exact CLI/package versions, passed
checksum/provenance, all three adapters and **9/9 installation transitions**:
cold install, unchanged duplicate rejection and offline forced recovery for
Codex, Claude and generic targets. Linux/macOS contribute another 18 transitions.
No runner build of the public CLI is permitted to stand in for this result.

Local PowerShell reproduction, with Python 3.12+, Go and an attestation-capable
GitHub CLI available:

```powershell
$env:CGO_ENABLED = '0'
go build -o bin/published-install-verifier.exe ./cmd/published-install-verifier
python scripts/check-published-install.py --sample instructions --cli-release v0.4.0-alpha.1 --verifier bin/published-install-verifier.exe --report output/published-install/windows/report.json
```

## Failure model and limits

The focused fixture suite runs on Linux and natively on Windows in PR CI:

```sh
python -m unittest discover -s scripts/tests -p 'test_published_windows.py'
```

It covers exact `.exe` asset selection; missing assets, wrong checksums and
failed provenance before execution; credential/environment isolation; all four
out-of-scope executable samples; paths with spaces and all three adapter
identities; payload substitution/extra files and forged receipt digests; and
five filesystem mutation classes (bytes, read-only/mode, timestamp, hard link,
extra directory). The native Windows case creates a real junction and requires
rejection without traversing its target. Host-mocked tests establish boundary
behavior, not Windows binary execution; only the hosted released-client report
establishes the latter.

This proves delivery into adapter targets, not actual Codex/Claude application
discovery or skill execution. Windows ARM64, Windows MCP/plugin startup, native
hook enforcement, provider/model readiness, full ACL handling and user-facing
upgrade/rollback/uninstall remain separate. #52 and epic #33 remain open.

## Main plugin baseline — 4 November 2026

PR #108 merged at `81fb1f6dac35595d2b3235710466f1aeaf021256`.
[Main CI](https://github.com/ai-knowledge-hub/ai-skills-guide/actions/runs/41823977109)
and Vercel passed. The
[published-plugin run on main](https://github.com/ai-knowledge-hub/ai-skills-guide/actions/runs/41824105080)
passed on both Linux and macOS, with candidate builds skipped. Both inspected
reports identify the actual `v0.4.0-alpha.1` CLI, the fixed plugin archive and
runtime contract, all five skill dependencies, receipts and registration.
Together they pass **24/24 installation-policy transitions and 12 first-use
probes** using Node 22.23.4, with advisory hooks, zero detected network attempts,
no executed proposed verification command and `merge_authorized: false`.

Windows implementation and hosted results are recorded separately below; this
baseline does not itself establish Windows execution.
