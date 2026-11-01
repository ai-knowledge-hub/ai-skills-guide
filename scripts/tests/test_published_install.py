import importlib.util
import contextlib
import io
import json
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
    def setUp(self):
        self.files = {"SKILL.md": b"reviewed instructions\n", "README.md": b"readme\n", "skill.yaml": b"manifest\n"}
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
            (directory / name).write_bytes(data)
            (directory / name).chmod(0o644)
        receipt = {"schema_version": "skills-hub.install-receipt/v1", "source": "remote", "module": "skills",
                   "id": "engineering/implementation-strategy", "version": "0.1.0", "runtime": "generic",
                   "artifact_sha256": self.artifact_sha}
        receipt.update(changes)
        (directory / check.RECEIPT).write_text(json.dumps(receipt))

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
            check.verify_installed(directory, self.files, "generic", self.artifact_sha)
            for changes in mutations:
                with self.subTest(changes=changes), self.assertRaises(ValueError):
                    self.install(directory, **changes)
                    check.verify_installed(directory, self.files, "generic", self.artifact_sha)

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
                        check.verify_installed(directory, self.files, "generic", self.artifact_sha)
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            binary = directory / "success-only"
            binary.write_text("#!/bin/sh\nexit 0\n")
            binary.chmod(0o755)
            with self.assertRaisesRegex(ValueError, "without an installed package"):
                check.verify_runtime(binary, check.REGISTRY, "generic", directory, self.files, self.artifact_sha, {})

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
        self.assertEqual(check.selected_release(index)[1]["version"], "0.1.0")
        for mutate in [lambda: entry.update(id="engineering/other"),
                       lambda: entry["versions"].pop(0),
                       lambda: entry["usability"].update(executable_helpers=[{"id": "helper"}])]:
            with self.subTest(mutate=mutate):
                original = json.loads(json.dumps(index))
                mutate()
                with self.assertRaises(ValueError):
                    check.selected_release(index)
                index = original
                entry = index["skills"][0]


if __name__ == "__main__":
    unittest.main()
