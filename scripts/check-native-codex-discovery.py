#!/usr/bin/env python3
"""Verify native Codex discovery of one exact published instruction release (#52)."""

import argparse
import datetime
import importlib.util
import json
from pathlib import Path
import platform
import shutil
import subprocess
import tarfile
import tempfile

import native_codex as native


spec = importlib.util.spec_from_file_location("published_check", Path(__file__).with_name("check-published-install.py"))
published = importlib.util.module_from_spec(spec)
spec.loader.exec_module(published)
ARTIFACT_SHA256 = "6e0946ab1e76fc8769171fa874f8cd4373e47655559a26c90cca4914efe5ab72"


def verify_case(binary, codex, root, registry_url, files, version, verifier, manifest, result):
    case = result["case"]
    environment, repo, target, explicit = native.case_environment(root, case)
    package = target / native.PACKAGE_ID
    instruction = package / "SKILL.md"
    expected_home = Path(environment.get("CODEX_HOME", str(root / "home/.codex")))
    result.update(target=str(target), explicit_target=explicit, probes=[])
    with native.AppServer(codex, repo, environment) as client:
        result["initialize"] = client.initialize(expected_home)

        def probe(state):
            before = published.filesystem_snapshot(package) if package.exists() else None
            discovery = client.skills(repo)
            if before is not None:
                native.require(published.filesystem_snapshot(package) == before, "Codex discovery mutated the installed package")
            scope = "repo" if case == "explicit-repository" else "user"
            result["probes"].append(native.assert_discovery(discovery, repo, instruction, state, scope))

        probe("baseline")
        command = [str(binary), "install", "--source", "remote", "--module", "skills", "--registry-url", registry_url,
                   "--entry", f"{native.PACKAGE_ID}@{published.VERSION}", "--runtime", "codex",
                   "--cache-dir", str(root / "cache")]
        if explicit:
            command += ["--target", str(target)]
        completed = subprocess.run(command, cwd=repo, env=environment, capture_output=True, timeout=150)
        native.require(completed.returncode == 0, "published installation failed: " + completed.stderr.decode(errors="replace")[-2000:])
        result["payload_sha256"] = published.verify_installed(package, files, "codex", version["sha256"],
                                                            verifier, manifest, version["manifest_sha256"])
        result["receipt_verified"] = True
        if case == "unregistered-target":
            probe("unregistered")
        else:
            probe("installed")
            instruction.rename(package / "removed-instruction.txt")
            probe("missing")
            (package / "removed-instruction.txt").unlink()
            instruction.write_text("---\nname: [\n---\nMalformed control; never execute.\n")
            probe("malformed")
            instruction.write_bytes(files["SKILL.md"])
            probe("restored")
            published.verify_installed(package, files, "codex", version["sha256"],
                                       verifier, manifest, version["manifest_sha256"])
        result["status"] = "passed"


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    source = parser.add_mutually_exclusive_group(required=True)
    source.add_argument("--cli-release", help="exact published installer tag, with checksums and signed provenance")
    source.add_argument("--cli", type=Path, help="explicit installer candidate; never labelled released")
    parser.add_argument("--codex-archive", type=Path, help="optional cached official archive; the same pinned hash is mandatory")
    parser.add_argument("--registry-url", default=published.REGISTRY)
    parser.add_argument("--verifier", type=Path, default=published.DEFAULT_VERIFIER)
    parser.add_argument("--report", type=Path, required=True)
    args = parser.parse_args(argv)
    report = {"schema_version": "native-codex-discovery/v1", "status": "failed",
              "checked_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
              "host": {"os": platform.system(), "architecture": platform.machine()},
              "installer_kind": "release" if args.cli_release else "candidate",
              "package": f"{native.PACKAGE_ID}@{published.VERSION}", "registry_url": args.registry_url,
              "codex": {}, "cases": [], "scope": "native discovery only; no model turn, skill execution or readiness promotion"}
    stage = "host"
    clean = None
    try:
        native.require((platform.system(), platform.machine().lower()) in native.ARCHIVES, "unsupported native discovery host")
        with tempfile.TemporaryDirectory(prefix="skills52-native space-") as temporary:
            clean = Path(temporary).resolve()
            bin_dir = clean / "bin"
            bin_dir.mkdir()
            stage = "installer"
            if args.cli_release:
                binary = published.released_cli(args.cli_release, bin_dir, report)
            else:
                native.require(args.cli.is_file(), "installer candidate is missing")
                binary = bin_dir / "skills-hub-candidate"
                shutil.copyfile(args.cli.resolve(), binary)
                binary.chmod(0o755)
            report["installer_sha256"] = native.digest(binary)
            native.require(args.verifier.is_file(), "build bin/published-install-verifier first")
            verifier = bin_dir / "published-install-verifier"
            shutil.copyfile(args.verifier.resolve(), verifier)
            verifier.chmod(0o755)
            report["verifier_sha256"] = native.digest(verifier)
            stage = "codex-client"
            codex = native.pinned_client(bin_dir, args.codex_archive, report["codex"])
            stage = "published-package"
            origin = published.https_origin(args.registry_url)
            registry = published.download(args.registry_url, origin)
            version = published.selected_release(json.loads(registry))
            native.require(version["sha256"] == ARTIFACT_SHA256, "published instruction artifact differs from the reviewed release")
            archive = published.download(version["artifact_url"], origin)
            manifest_bytes = published.download(version["manifest_url"], origin)
            native.require(published.sha256(archive) == version["sha256"], "published archive checksum mismatch")
            native.require(published.sha256(manifest_bytes) == version["manifest_sha256"], "published manifest checksum mismatch")
            files = published.archive_files(archive)
            native.require(files["skill.yaml"] == manifest_bytes, "served manifest differs from archive manifest")
            pinned = clean / "pinned-package"
            for name, data in files.items():
                path = pinned / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(data)
            manifest = pinned / "skill.yaml"
            report["pinned_manifest"] = published.inspect_pinned(verifier, manifest, version["manifest_sha256"])
            native.require("codex" in report["pinned_manifest"]["runtimes"], "pinned package does not declare Codex")
            report.update(registry_sha256=published.sha256(registry), selected_release=version,
                          published_files={name: published.sha256(data) for name, data in sorted(files.items())})
            for case in native.CASES:
                stage = case
                result = {"case": case, "status": "failed"}
                report["cases"].append(result)
                root = clean / case
                verify_case(binary, codex, root, args.registry_url, files, version, verifier, manifest, result)
            report["status"] = "passed"
    except (OSError, ValueError, KeyError, TypeError, AttributeError, tarfile.TarError, subprocess.SubprocessError) as error:
        report.update(failure_stage=stage, error=str(error))
    if clean is not None:
        # Preserve exact path relationships in success and failure reports.
        report = json.loads(json.dumps(report).replace(str(clean), "$WORK"))
    args.report.parent.mkdir(parents=True, exist_ok=True)
    args.report.write_text(json.dumps(report, indent=2) + "\n")
    print(f"{report['status']}: native Codex discovery, {report['installer_kind']} installer; report: {args.report}")
    if report["status"] != "passed":
        print(report["error"])
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
