"""Windows delivery boundaries; run natively as well as on POSIX hosts.

Use fixed release names, retained archive bytes and an independent receipt tree
model. Mocked cross-platform cases are not Windows execution evidence.
"""

import contextlib
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import stat
import subprocess
import tempfile
import unittest
from unittest.mock import patch


ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("windows_delivery_check", ROOT / "scripts/check-published-install.py")
check = importlib.util.module_from_spec(spec)
spec.loader.exec_module(check)
TAG = "v0.4.0-alpha.1"
WINDOWS_EXE = "skills-hub_v0.4.0-alpha.1_windows_amd64.exe"


class WindowsDeliveryTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.build = tempfile.TemporaryDirectory(prefix="windows oracle space-")
        cls.verifier = Path(cls.build.name) / ("verifier.exe" if os.name == "nt" else "verifier")
        subprocess.run(["go", "build", "-o", str(cls.verifier), "./cmd/published-install-verifier"], cwd=ROOT,
                       env={**os.environ, "CGO_ENABLED": "0"}, check=True)
        release = ROOT / "releases/skills/engineering/implementation-strategy/0.1.0"
        cls.archive = (release / "package.tar.gz").read_bytes()
        cls.files = check.archive_files(cls.archive)
        cls.artifact = hashlib.sha256(cls.archive).hexdigest()
        cls.manifest_sha = hashlib.sha256(cls.files["skill.yaml"]).hexdigest()

    @classmethod
    def tearDownClass(cls):
        cls.build.cleanup()

    def fixture(self, root):
        root.mkdir(parents=True, exist_ok=True)
        for name, data in self.files.items():
            path = root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(data)
            path.chmod(0o644)
        digest = hashlib.sha256()
        for path in sorted(root.rglob("*")):
            name = path.relative_to(root).as_posix()
            kind, mode = ("dir", "755") if path.is_dir() else ("file", "644")
            digest.update(f"{kind}\0{name}\0{mode}\0".encode())
            if path.is_file():
                digest.update(path.read_bytes())
            digest.update(b"\0")
        receipt = {"schema_version": "skills-hub.install-receipt/v1", "source": "remote", "module": "skills",
                   "id": "engineering/implementation-strategy", "version": "0.1.0", "runtime": "generic",
                   "artifact_sha256": self.artifact,
                   "runtime_contract_sha256": "695b5bccd308e33ab968c5673c3fc37b84ad7d18af13a85e141e3dc627cda6b0",
                   "tree_sha256": digest.hexdigest(), "installed_at": "2026-11-01T00:00:00Z"}
        (root / check.RECEIPT).write_text(json.dumps(receipt))

    def verify(self, root):
        return check.verify_installed(root, self.files, "generic", self.artifact, self.verifier,
                                      root / "skill.yaml", self.manifest_sha)

    def test_exact_published_names_and_unknown_hosts(self):
        expected = [("Windows", "AMD64", WINDOWS_EXE),
                    ("Windows", "ARM64", "skills-hub_v0.4.0-alpha.1_windows_arm64.exe"),
                    ("Linux", "x86_64", "skills-hub_v0.4.0-alpha.1_linux_amd64"),
                    ("Darwin", "arm64", "skills-hub_v0.4.0-alpha.1_darwin_arm64")]
        for system, machine, name in expected:
            with self.subTest(system=system, machine=machine):
                self.assertEqual(check.release_asset_names(TAG, system, machine),
                                 (name, "SHA256SUMS", "skills-hub_v0.4.0-alpha.1_provenance.json"))
        for system, machine in [("Windows", "i386"), ("FreeBSD", "amd64")]:
            with self.subTest(system=system, machine=machine), self.assertRaises(ValueError):
                check.release_asset_names(TAG, system, machine)

    def release_responses(self, bad=None):
        names = [WINDOWS_EXE, "SHA256SUMS", "skills-hub_v0.4.0-alpha.1_provenance.json"]
        if bad == "missing-exe":
            names[0] = WINDOWS_EXE.removesuffix(".exe")
        metadata = {"tag_name": TAG, "draft": False, "assets": [
            {"name": name, "browser_download_url": f"https://github.com/ai-knowledge-hub/ai-skills-guide/releases/download/{TAG}/{name}"}
            for name in names]}
        binary = b"fixture PE binary; must not execute"
        digest = "0" * 64 if bad == "checksum" else hashlib.sha256(binary).hexdigest()
        return [json.dumps(metadata).encode(), binary, f"{digest}  {names[0]}\n".encode(), b"fixture provenance"]

    def test_windows_binary_requires_exact_asset_checksum_and_provenance(self):
        for bad in ["missing-exe", "checksum", "provenance", None]:
            with self.subTest(bad=bad), tempfile.TemporaryDirectory() as temporary, \
                    patch.object(check.platform, "system", return_value="Windows"), \
                    patch.object(check.platform, "machine", return_value="AMD64"), \
                    patch.object(check, "download", side_effect=self.release_responses(bad)), \
                    patch.object(check.subprocess, "run") as run:
                report = {}
                if bad == "provenance":
                    run.side_effect = subprocess.CalledProcessError(1, ["gh", "attestation", "verify"])
                if bad:
                    with self.assertRaises((ValueError, subprocess.CalledProcessError)):
                        check.released_cli(TAG, Path(temporary), report)
                    self.assertNotIn("checksum_and_provenance", report["release"])
                else:
                    binary = check.released_cli(TAG, Path(temporary), report)
                    self.assertEqual(binary.name, WINDOWS_EXE)
                    self.assertEqual(report["release"]["checksum_and_provenance"], "passed")
                self.assertEqual(run.call_count, 0 if bad in {"missing-exe", "checksum"} else 1)
                if run.called:
                    command = run.call_args.args[0]
                    self.assertEqual(command[:3], ["gh", "attestation", "verify"])
                    self.assertEqual(command[3], str(Path(temporary) / WINDOWS_EXE))
                    self.assertIn("refs/tags/v0.4.0-alpha.1", command)
                    self.assertIn("ai-knowledge-hub/ai-skills-guide/.github/workflows/release-on-tag.yml", command)
                    self.assertIn("--deny-self-hosted-runners", command)

    def test_windows_env_keeps_only_os_context_and_disposable_temp(self):
        toxic = {"SYSTEMROOT": r"C:\Windows", "PATH": "hostile-path", "GH_TOKEN": "not-a-real-token",
                 "HOME": "real-home", "USERPROFILE": "real-profile", "APPDATA": "real-appdata",
                 "CODEX_HOME": "real-codex", "CLAUDE_HOME": "real-claude", "NODE_OPTIONS": "--require injected",
                 "TEMP": "real-temp", "TMP": "real-tmp", "HTTPS_PROXY": "real-proxy"}
        with tempfile.TemporaryDirectory(prefix="child space-") as temporary, \
                patch.dict(os.environ, toxic, clear=True), patch.object(check.platform, "system", return_value="Windows"):
            work = Path(temporary)
            environment = check.clean_environment(work)
            self.assertEqual(environment, {"SYSTEMROOT": r"C:\Windows", "PATH": r"C:\Windows\System32",
                                           "TEMP": str(work / "process-temp"), "TMP": str(work / "process-temp")})
            self.assertTrue((work / "process-temp").is_dir())
            self.assertEqual(set(check.clean_environment()), {"SYSTEMROOT", "PATH"})
        with patch.dict(os.environ, {}, clear=True), patch.object(check.platform, "system", return_value="Windows"):
            with self.assertRaisesRegex(ValueError, "SYSTEMROOT"):
                check.clean_environment()

    def test_windows_rejects_every_executable_sample_before_download_or_execution(self):
        for sample in ["ga4", "bigquery", "meta-ads", "code-maintenance"]:
            with self.subTest(sample=sample), tempfile.TemporaryDirectory() as temporary:
                report = Path(temporary) / "report.json"
                with patch("sys.argv", ["check", "--sample", sample, "--cli-release", TAG, "--report", str(report)]), \
                        patch.object(check.platform, "system", return_value="Windows"), \
                        patch.object(check, "download") as download, patch.object(check.subprocess, "run") as run, \
                        contextlib.redirect_stdout(io.StringIO()):
                    self.assertEqual(check.main(), 1)
                evidence = json.loads(report.read_text())
                self.assertEqual(evidence["failure_stage"], "host")
                self.assertEqual(evidence["status"], "failed")
                self.assertEqual(evidence["client_kind"], "release")
                download.assert_not_called()
                run.assert_not_called()

    def test_native_receipt_and_bytes_in_space_paths(self):
        with tempfile.TemporaryDirectory(prefix="payload oracle space-") as temporary:
            root = Path(temporary) / "runtime space/skills/engineering/implementation-strategy"
            self.fixture(root)
            self.verify(root)
            for mutation in ["bytes", "extra", "receipt-artifact", "receipt-tree"]:
                with self.subTest(mutation=mutation):
                    shutil.rmtree(root)
                    self.fixture(root)
                    if mutation == "bytes":
                        (root / "SKILL.md").write_bytes(b"substituted bytes")
                    elif mutation == "extra":
                        (root / "extra.exe").write_bytes(b"undeclared executable")
                    else:
                        path = root / check.RECEIPT
                        receipt = json.loads(path.read_text())
                        receipt["artifact_sha256" if mutation == "receipt-artifact" else "tree_sha256"] = "0" * 64
                        path.write_text(json.dumps(receipt))
                    with self.assertRaises(ValueError):
                        self.verify(root)

    def test_snapshot_detects_bytes_metadata_and_membership_mutations(self):
        # The same state relation is checked natively on NTFS and POSIX. In
        # particular Windows chmod changes read-only state rather than rwx bits.
        for mutation in ["bytes", "mode", "mtime", "hardlink", "empty-directory"]:
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary) / "package"
                self.fixture(root)
                before = check.filesystem_snapshot(root)
                path = root / "SKILL.md"
                if mutation == "bytes":
                    path.write_bytes(b"changed")
                elif mutation == "mode":
                    path.chmod(stat.S_IREAD)
                elif mutation == "mtime":
                    info = path.stat()
                    os.utime(path, ns=(info.st_atime_ns, info.st_mtime_ns + 2_000_000_000))
                elif mutation == "hardlink":
                    (Path(temporary) / "outside-link").hardlink_to(path)
                else:
                    (root / "unexpected-empty").mkdir()
                self.assertNotEqual(before, check.filesystem_snapshot(root))
                path.chmod(0o644)

    @unittest.skipUnless(os.name == "nt", "requires actual Windows junction semantics")
    def test_native_junction_is_rejected_without_traversing_target(self):
        with tempfile.TemporaryDirectory(prefix="junction space-") as temporary:
            directory = Path(temporary)
            root, outside = directory / "package", directory / "outside"
            self.fixture(root)
            outside.mkdir()
            (outside / "private.txt").write_bytes(b"must not read")
            junction = root / "redirect"
            subprocess.run(["cmd.exe", "/d", "/c", "mklink", "/J", str(junction), str(outside)],
                           check=True, capture_output=True)
            try:
                # is_symlink alone misses junctions. The snapshot must retain
                # the redirect metadata without following or reading its target.
                snapshot = check.filesystem_snapshot(root)
                self.assertIn("redirect", snapshot)
                self.assertNotIn("redirect/private.txt", snapshot)
                with self.assertRaisesRegex(ValueError, "reparse point"):
                    self.verify(root)
                with self.assertRaisesRegex(ValueError, "link/reparse point"):
                    check.filesystem_snapshot(junction)
            finally:
                junction.rmdir()

    def test_all_adapters_keep_exe_names_and_argv_paths_outside_checkout(self):
        version = {"version": "0.1.0", "sha256": self.artifact, "manifest_sha256": self.manifest_sha,
                   "artifact_url": "https://example.com/package.tar.gz", "manifest_url": "https://example.com/skill.yaml"}
        index = {"registry_version": "1.3", "skills": [{"id": "engineering/implementation-strategy", "versions": [version]}]}
        with tempfile.TemporaryDirectory(prefix="candidate space-") as temporary:
            candidate = Path(temporary) / "candidate.exe"
            candidate.write_bytes(b"orchestration fixture; never executed")
            report = Path(temporary) / "evidence.json"
            seen = []

            def verify(binary, registry, runtime, directory, *args):
                self.assertEqual(binary.name, "skills-hub-candidate.exe")
                self.assertIn(" ", str(directory))
                self.assertFalse(directory.is_relative_to(ROOT))
                self.assertEqual(args[3].name, "published-install-verifier.exe")
                self.assertTrue(binary.is_file() and args[3].is_file())
                seen.append(runtime)

            # Only naming/orchestration is simulated. Native Go receipt checking
            # is exercised above and real released-client execution runs in CI.
            with patch("sys.argv", ["check", "--cli", str(candidate), "--verifier", str(self.verifier), "--report", str(report)]), \
                    patch.object(check.platform, "system", return_value="Windows"), \
                    patch.object(check, "download", side_effect=[json.dumps(index).encode(), self.archive, self.files["skill.yaml"]]), \
                    patch.object(check, "inspect_pinned", return_value={"runtimes": ["codex", "claude", "generic"]}), \
                    patch.object(check, "verify_runtime", side_effect=verify), contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(check.main(), 0)
            self.assertEqual(seen, ["codex", "claude", "generic"])
            evidence = json.loads(report.read_text())
            self.assertEqual(evidence["client_kind"], "candidate")
            self.assertNotIn("release", evidence)


if __name__ == "__main__":
    unittest.main()
