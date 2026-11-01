import importlib.util
import contextlib
import io
import json
import hashlib
import os
from pathlib import Path
import tarfile
import subprocess
import tempfile
import unittest
from unittest.mock import patch


spec = importlib.util.spec_from_file_location("published_check", Path(__file__).parents[1] / "check-published-install.py")
check = importlib.util.module_from_spec(spec)
spec.loader.exec_module(check)


class PublishedInstallCheckTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.build = tempfile.TemporaryDirectory(prefix="receipt-verifier-test-")
        cls.verifier = Path(cls.build.name) / "verifier"
        root = Path(__file__).resolve().parents[2]
        subprocess.run(["go", "build", "-o", str(cls.verifier), "./cmd/published-install-verifier"], cwd=root, check=True)
        cls.release_dir = root / "releases/skills/engineering/implementation-strategy/0.1.0"

    @classmethod
    def tearDownClass(cls):
        cls.build.cleanup()

    def setUp(self):
        self.files = check.archive_files((self.release_dir / "package.tar.gz").read_bytes())
        self.manifest_sha = check.sha256(self.files["skill.yaml"])
        self.artifact_sha = "a" * 64

    def archive(self, extra=None):
        output = io.BytesIO()
        with tarfile.open(fileobj=output, mode="w:gz") as archive:
            for name, data in self.files.items():
                info = tarfile.TarInfo(name)
                info.size = len(data)
                info.mode = 0o644
                archive.addfile(info, io.BytesIO(data))
            if extra:
                archive.addfile(extra)
        return output.getvalue()

    def install(self, directory, **changes):
        for name, data in self.files.items():
            (directory / name).parent.mkdir(parents=True, exist_ok=True)
            (directory / name).write_bytes(data)
            (directory / name).chmod(0o644)
        receipt = {"schema_version": "skills-hub.install-receipt/v1", "source": "remote", "module": "skills",
                   "id": "engineering/implementation-strategy", "version": "0.1.0", "runtime": "generic",
                   "artifact_sha256": self.artifact_sha,
                   # Contract digest independently retained from the published
                   # 0.1.0 installation observed before this verifier existed.
                   "runtime_contract_sha256": "695b5bccd308e33ab968c5673c3fc37b84ad7d18af13a85e141e3dc627cda6b0",
                   "tree_sha256": self.tree_digest(directory),
                   "installed_at": "2026-11-01T00:00:00Z"}
        receipt.update(changes)
        (directory / check.RECEIPT).write_text(json.dumps(receipt))

    def tree_digest(self, directory):
        digest = hashlib.sha256()
        for path in sorted(directory.rglob("*")):
            name = path.relative_to(directory).as_posix()
            if name == check.RECEIPT:
                continue
            kind, mode = ("dir", "755") if path.is_dir() else ("file", "644")
            digest.update(f"{kind}\0{name}\0{mode}\0".encode())
            if path.is_file():
                digest.update(path.read_bytes())
            digest.update(b"\0")
        return digest.hexdigest()

    def verify(self, directory):
        return check.verify_installed(directory, self.files, "generic", self.artifact_sha,
                                      self.verifier, directory / "skill.yaml", self.manifest_sha)

    def test_archive_oracle_rejects_links_duplicates_paths_and_executables(self):
        self.assertEqual(check.archive_files(self.archive()), self.files)
        for name, kind, mode in [("../outside", tarfile.REGTYPE, 0o644), ("SKILL.md", tarfile.REGTYPE, 0o644),
                                 ("link", tarfile.SYMTYPE, 0o644), ("run.sh", tarfile.REGTYPE, 0o755)]:
            with self.subTest(name=name, kind=kind), self.assertRaises(ValueError):
                info = tarfile.TarInfo(name)
                info.type, info.mode = kind, mode
                check.archive_files(self.archive(info))

    def test_receipt_cannot_substitute_identity_runtime_source_or_artifact(self):
        mutations = [{"id": "engineering/another-skill"}, {"version": "0.2.0"}, {"module": "tools"},
                     {"runtime": "codex"}, {"source": "local"}, {"artifact_sha256": "b" * 64},
                     {"schema_version": "unknown"}]
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            self.install(directory)
            self.verify(directory)
            for changes in mutations:
                with self.subTest(changes=changes), self.assertRaises(ValueError):
                    self.install(directory, **changes)
                    self.verify(directory)

    def test_receipt_or_exit_success_is_not_a_payload_oracle(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            self.install(directory)
            for mutation in ["missing", "changed", "extra", "link", "executable"]:
                with self.subTest(mutation=mutation):
                    self.install(directory)
                    extra = directory / "extra"
                    if extra.exists() or extra.is_symlink():
                        extra.unlink()
                    if mutation == "missing":
                        (directory / "SKILL.md").unlink()
                    elif mutation == "changed":
                        (directory / "SKILL.md").write_text("substituted")
                    elif mutation == "extra":
                        extra.write_text("undeclared")
                    elif mutation == "link":
                        extra.symlink_to(directory / "SKILL.md")
                    else:
                        (directory / "SKILL.md").chmod(0o755)
                    with self.assertRaises(ValueError):
                        self.verify(directory)
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            binary = directory / "success-only"
            binary.write_text("#!/bin/sh\nexit 0\n")
            binary.chmod(0o755)
            with self.assertRaisesRegex(ValueError, "without an installed package"):
                check.verify_runtime(binary, check.REGISTRY, "generic", directory, self.files, self.artifact_sha, {},
                                     self.verifier, self.release_dir / "skill.yaml", self.manifest_sha)

    def test_complete_receipt_metadata_must_pass_the_runtime_consumer(self):
        mutations = [
            {"tree_sha256": None}, {"tree_sha256": "invalid"}, {"tree_sha256": "0" * 64},
            {"runtime_contract_sha256": None}, {"runtime_contract_sha256": "invalid"},
            {"runtime_contract_sha256": "0" * 64}, {"installed_at": None},
            {"installed_at": "not-a-timestamp"}, {"installed_at": "0001-01-01T00:00:00Z"},
            {"installed_at": 123}, {"unexpected_field": True},
        ]
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            self.install(directory)
            self.verify(directory)
            for changes in mutations:
                with self.subTest(changes=changes):
                    self.install(directory)
                    receipt_path = directory / check.RECEIPT
                    receipt = json.loads(receipt_path.read_text())
                    for key, value in changes.items():
                        if value is None:
                            receipt.pop(key, None)
                        else:
                            receipt[key] = value
                    receipt_path.write_text(json.dumps(receipt))
                    with self.assertRaises(ValueError):
                        self.verify(directory)

    def test_duplicate_metadata_mutations_fail_before_repair(self):
        for mutation in ["file-mode", "directory-mode", "symlink", "extra-directory", "file-type", "mtime", "hardlink"]:
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as temporary:
                directory = Path(temporary)
                pinned = directory / "pinned"
                pinned.mkdir()
                self.install(pinned)
                self.verify(pinned)
                installed = directory / "generic/runtime/skills" / check.PACKAGE
                calls = []

                def fake_install(*args, **kwargs):
                    calls.append(args[0])
                    if len(calls) == 1:
                        installed.mkdir(parents=True)
                        self.install(installed)
                        return subprocess.CompletedProcess(args[0], 0, b"", b"")
                    if mutation == "file-mode":
                        (installed / "SKILL.md").chmod(0o755)
                    elif mutation == "directory-mode":
                        (installed / "examples").chmod(0o700)
                    elif mutation == "symlink":
                        target = directory / "linked-readme"
                        target.write_bytes((installed / "README.md").read_bytes())
                        (installed / "README.md").unlink()
                        (installed / "README.md").symlink_to(target)
                    elif mutation == "extra-directory":
                        (installed / "unexpected-empty").mkdir()
                    elif mutation == "mtime":
                        path = installed / "SKILL.md"
                        info = path.stat()
                        os.utime(path, ns=(info.st_atime_ns, info.st_mtime_ns + 1_000_000_000))
                    elif mutation == "hardlink":
                        (directory / "outside-hardlink").hardlink_to(installed / "SKILL.md")
                    else:
                        (installed / "README.md").unlink()
                        (installed / "README.md").mkdir()
                    return subprocess.CompletedProcess(args[0], 1, b"", b"duplicate rejected")

                # The first valid installation has already been checked against
                # the real Go consumer above; isolate duplicate effects here.
                with patch.object(check.subprocess, "run", side_effect=fake_install), \
                        patch.object(check, "inspect_pinned", return_value={"receipt_verified": True}):
                    with self.assertRaisesRegex(ValueError, "duplicate install did not reject safely"):
                        check.verify_runtime(directory / "cli", check.REGISTRY, "generic", directory,
                                             self.files, self.artifact_sha, {}, self.verifier,
                                             pinned / "skill.yaml", self.manifest_sha)
                self.assertEqual(len(calls), 2, "repair must not hide duplicate-attempt mutation")

    def test_release_with_no_binary_blocks_before_any_execution(self):
        metadata = {"tag_name": "v0.3.0-alpha.1", "draft": False, "assets": []}
        with tempfile.TemporaryDirectory() as temporary, patch.object(check, "download", return_value=json.dumps(metadata).encode()), \
                patch.object(check.subprocess, "run") as run:
            with self.assertRaisesRegex(ValueError, "published CLI assets missing"):
                check.released_cli("v0.3.0-alpha.1", Path(temporary), {})
            run.assert_not_called()

    def test_checksum_mismatch_blocks_before_provenance_or_execution(self):
        tag = "v1.0.0"
        names = check.release_asset_names(tag, "Linux", "x86_64")
        metadata = {"tag_name": tag, "draft": False, "assets": [
            {"name": name, "browser_download_url": f"https://github.com/{check.REPOSITORY}/releases/download/{tag}/{name}"}
            for name in names]}
        responses = [json.dumps(metadata).encode(), b"untrusted binary", f"{'0' * 64}  {names[0]}\n".encode(), b"{}"]
        with tempfile.TemporaryDirectory() as temporary, patch.object(check.platform, "system", return_value="Linux"), \
                patch.object(check.platform, "machine", return_value="x86_64"), \
                patch.object(check, "download", side_effect=responses), patch.object(check.subprocess, "run") as run:
            with self.assertRaisesRegex(ValueError, "checksum mismatch"):
                check.released_cli(tag, Path(temporary), {})
            run.assert_not_called()
            self.assertEqual((Path(temporary) / names[0]).stat().st_mode & 0o111, 0)

    def test_invalid_provenance_keeps_released_binary_non_executable(self):
        tag = "v1.0.0"
        names = check.release_asset_names(tag, "Linux", "x86_64")
        metadata = {"tag_name": tag, "draft": False, "assets": [
            {"name": name, "browser_download_url": f"https://github.com/{check.REPOSITORY}/releases/download/{tag}/{name}"}
            for name in names]}
        binary = b"untrusted binary"
        responses = [json.dumps(metadata).encode(), binary, f"{check.sha256(binary)}  {names[0]}\n".encode(), b"{}"]
        with tempfile.TemporaryDirectory() as temporary, patch.object(check.platform, "system", return_value="Linux"), \
                patch.object(check.platform, "machine", return_value="x86_64"), \
                patch.object(check, "download", side_effect=responses), \
                patch.object(check.subprocess, "run", side_effect=subprocess.CalledProcessError(1, ["gh", "attestation", "verify"])):
            with self.assertRaises(subprocess.CalledProcessError):
                check.released_cli(tag, Path(temporary), {})
            self.assertEqual((Path(temporary) / names[0]).stat().st_mode & 0o111, 0)

    def test_malformed_endpoint_retains_failed_candidate_report(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            binary = directory / "candidate"
            binary.write_text("not invoked")
            report = directory / "evidence.json"
            with patch("sys.argv", ["check", "--cli", str(binary), "--report", str(report)]), \
                    patch.object(check, "download", return_value=b"[]"), contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(check.main(), 1)
            evidence = json.loads(report.read_text())
            self.assertEqual(evidence["status"], "failed")
            self.assertEqual(evidence["client_kind"], "candidate")
            self.assertEqual(evidence["failure_stage"], "published-release")

    def test_empty_release_tag_fails_as_release_without_candidate_fallback(self):
        with tempfile.TemporaryDirectory() as temporary:
            report = Path(temporary) / "evidence.json"
            with patch("sys.argv", ["check", "--cli-release", "", "--report", str(report)]), \
                    patch.object(check, "download") as download, contextlib.redirect_stdout(io.StringIO()):
                self.assertEqual(check.main(), 1)
            evidence = json.loads(report.read_text())
            self.assertEqual(evidence["client_kind"], "release")
            self.assertEqual(evidence["failure_stage"], "client")
            download.assert_not_called()

    def test_exact_release_and_https_boundaries(self):
        for tag in ["latest", "v1", "v1.0.0/other", "$(command)"]:
            with self.subTest(tag=tag), self.assertRaises(ValueError):
                check.release_asset_names(tag, "Linux", "x86_64")
        for url in ["http://example.com/registry.json", "https://user:secret@example.com/r", "https://example.com/r#fragment"]:
            with self.subTest(url=url), self.assertRaises(ValueError):
                check.https_origin(url)
        with self.assertRaisesRegex(ValueError, "same HTTPS origin"):
            check.download("https://other.example/asset", check.https_origin("https://example.com/registry.json"))

    def test_pinned_selection_cannot_follow_latest_or_another_package(self):
        entry = {"id": check.PACKAGE, "latest": "9.0.0", "usability": {"execution": "instructions", "availability": "documentation-only"},
                 "versions": [{"version": "0.1.0", "sha256": "a" * 64, "manifest_sha256": "b" * 64}, {"version": "9.0.0"}]}
        index = {"registry_version": "1.3", "skills": [entry]}
        self.assertEqual(check.selected_release(index)["version"], "0.1.0")
        for mutate in [lambda: entry.update(id="engineering/other"),
                       lambda: entry["versions"].pop(0),
                       lambda: entry["versions"].append(entry["versions"][0])]:
            with self.subTest(mutate=mutate):
                original = json.loads(json.dumps(index))
                mutate()
                with self.assertRaises(ValueError):
                    check.selected_release(index)
                index = original
                entry = index["skills"][0]

    def run_pinned_case(self, current, manifest=None, runtimes=None):
        if manifest is not None:
            self.files["skill.yaml"] = manifest
        archive = self.archive()
        pinned = {"version": check.VERSION, "sha256": check.sha256(archive),
                  "manifest_sha256": check.sha256(self.files["skill.yaml"]),
                  "artifact_url": "https://example.com/archive.tar.gz",
                  "manifest_url": "https://example.com/skill.yaml"}
        index = {"registry_version": "1.3", "skills": [
            {"id": check.PACKAGE, "latest": "9.0.0", "versions": [pinned, {"version": "9.0.0"}], **current}]}
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            binary = directory / "candidate"
            binary.write_text("not executed in this orchestration test")
            report = directory / "report.json"
            args = ["check", "--cli", str(binary), "--verifier", str(self.verifier), "--report", str(report)]
            if runtimes:
                args += ["--runtimes", *runtimes]
            with patch("sys.argv", args), patch.object(check, "download", side_effect=[
                    json.dumps(index).encode(), archive, self.files["skill.yaml"]]), \
                    patch.object(check, "verify_runtime") as verify, contextlib.redirect_stdout(io.StringIO()):
                code = check.main()
            return code, json.loads(report.read_text()), verify.call_count

    def test_current_release_declarations_do_not_classify_pinned_subject(self):
        code, report, calls = self.run_pinned_case({
            "runtimes": ["future-runtime"],
            "usability": {"execution": "remote-integration", "availability": "not-verified",
                          "executable_helpers": [{"id": "new-release-helper"}]},
        })
        self.assertEqual(code, 0)
        self.assertEqual(calls, 3)
        self.assertEqual(report["pinned_manifest"]["runtimes"], ["codex", "claude", "generic"])

    def test_verifier_rejects_substituted_pinned_manifest_or_digest(self):
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            self.install(directory)
            path = directory / "skill.yaml"
            with self.assertRaisesRegex(ValueError, "pinned manifest digest mismatch"):
                check.inspect_pinned(self.verifier, path, "0" * 64)
            altered = path.read_bytes().replace(check.PACKAGE.encode(), b"engineering/another-skill")
            path.write_bytes(altered)
            with self.assertRaisesRegex(ValueError, "does not match the pinned subject"):
                check.inspect_pinned(self.verifier, path, check.sha256(altered))

    def test_pinned_manifest_runtime_controls_selection(self):
        manifest = self.files["skill.yaml"].replace(b"  - codex\n  - claude\n", b"")
        code, report, calls = self.run_pinned_case({"runtimes": ["codex", "generic"]}, manifest, ["codex"])
        self.assertEqual(code, 1)
        self.assertEqual(calls, 0)
        self.assertIn("pinned sample does not declare codex", report["error"])

    def test_pinned_manifest_usability_controls_instruction_sample(self):
        manifest = self.files["skill.yaml"].replace(b"availability: documentation-only", b"availability: not-verified")
        code, report, calls = self.run_pinned_case({
            "usability": {"availability": "documentation-only", "execution": "instructions"}}, manifest)
        self.assertEqual(code, 1)
        self.assertEqual(calls, 0)
        self.assertIn("pinned sample must be instruction-only", report["error"])


if __name__ == "__main__":
    unittest.main()
