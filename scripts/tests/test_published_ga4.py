"""Test the GA4 evidence checker with retained bytes and bounded local processes."""

import contextlib
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tarfile
import tempfile
import unittest
from unittest.mock import patch


ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("published_check_ga4", ROOT / "scripts/check-published-install.py")
check = importlib.util.module_from_spec(spec)
spec.loader.exec_module(check)


class PublishedGA4Tests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.build = tempfile.TemporaryDirectory(prefix="ga4-checker-tests-")
        cls.verifier = Path(cls.build.name) / "verifier"
        cls.binary = Path(cls.build.name) / "skills-hub"
        environment = {**os.environ, "CGO_ENABLED": "0", "GOFLAGS": "", "GOWORK": "off"}
        for output, package, tags in [(cls.verifier, "published-install-verifier", []),
                                      (cls.binary, "skills-hub", ["-tags", "installer_release"])]:
            subprocess.run(["go", "build", *tags, "-o", str(output), f"./cmd/{package}"],
                           cwd=ROOT, env=environment, check=True)
        cls.archive = (ROOT / "releases/tools-mcp/analytics/ga4-mcp-connector/0.2.0/package.tar.gz").read_bytes()
        cls.modes = {}
        cls.files = check.archive_files(cls.archive, check.GA4, cls.modes)
        cls.manifest_sha = check.sha256(cls.files["tool.yaml"])
        cls.artifact_sha = check.sha256(cls.archive)
        cls.guide = subprocess.check_output([str(cls.binary), "help", "setup"], text=True)

    @classmethod
    def tearDownClass(cls):
        cls.build.cleanup()

    def package(self, directory, files=None, **changes):
        files = self.files if files is None else files
        directory.mkdir(parents=True, exist_ok=True)
        for name, data in files.items():
            path = directory / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(data)
            path.chmod(self.modes[name])
        for name, mode in self.modes.items():
            (directory / name).chmod(mode)
        digest = hashlib.sha256()
        for path in sorted(directory.rglob("*")):
            if path.name == check.RECEIPT:
                continue
            kind, mode = ("dir", "755") if path.is_dir() else ("file", format(path.stat().st_mode & 0o777, "o"))
            digest.update(f"{kind}\0{path.relative_to(directory).as_posix()}\0{mode}\0".encode())
            if path.is_file():
                digest.update(path.read_bytes())
            digest.update(b"\0")
        receipt = {"schema_version": "skills-hub.install-receipt/v1", "source": "remote", "module": "tools",
                   "id": "analytics/ga4-mcp-connector", "version": "0.2.0", "runtime": "generic",
                   "artifact_sha256": self.artifact_sha,
                   # Golden observed from the actual v0.4.0-alpha.1 GA4 install,
                   # not calculated by the verifier under test.
                   "runtime_contract_sha256": "d1032674f1ae1d2687860cfd3a66ebd5bc945363a26676b0cd67987fee7a39a4",
                   "tree_sha256": digest.hexdigest(), "installed_at": "2026-11-03T00:00:00Z"}
        receipt.update(changes)
        (directory / check.RECEIPT).write_text(json.dumps(receipt))
        return directory

    def verify(self, package, files=None):
        return check.verify_installed(package, self.files if files is None else files, "generic", self.artifact_sha,
                                      self.verifier, package / "tool.yaml", self.manifest_sha, check.GA4, self.modes)

    def startup(self, directory, files=None):
        package = self.package(directory / "package", files)
        work = directory / "outside-package"
        work.mkdir()
        return check.ga4_startup.verify(self.binary, b"install.setup_guidance: skills-hub help setup\n", package, work)

    def test_selection_pins_ga4_and_ignores_newer_release_declarations(self):
        pinned = {"version": "0.2.0", "sha256": self.artifact_sha, "manifest_sha256": self.manifest_sha}
        entry = {"id": "analytics/ga4-mcp-connector", "latest": "99.0.0", "versions": [pinned, {"version": "99.0.0"}],
                 "usability": {"availability": "template-only"}, "runtimes": []}
        index = {"registry_version": "1.3", "skills": [entry]}
        self.assertEqual(check.selected_release(index, check.GA4), pinned)
        for versions in [[{"version": "99.0.0"}], [pinned, pinned]]:
            with self.subTest(versions=versions), self.assertRaises(ValueError):
                check.selected_release({"registry_version": "1.3", "skills": [{**entry, "versions": versions}]}, check.GA4)
        with self.assertRaises(ValueError):
            check.selected_release(index)  # GA4 cannot become the instruction-only case.

    def test_tool_archive_does_not_relax_instruction_or_path_safety(self):
        with self.assertRaises(ValueError):
            check.archive_files(self.archive)
        for name, kind in [("../escape", tarfile.REGTYPE), ("scripts/link", tarfile.SYMTYPE),
                           ("tool.yaml", tarfile.REGTYPE), ("README.md", tarfile.DIRTYPE)]:
            output = io.BytesIO()
            with tarfile.open(fileobj=output, mode="w:gz") as archive:
                for path, data in self.files.items():
                    info = tarfile.TarInfo(path)
                    info.size = len(data)
                    archive.addfile(info, io.BytesIO(data))
                info = tarfile.TarInfo(name)
                info.type = kind
                archive.addfile(info)
            with self.subTest(name=name), self.assertRaises(ValueError):
                check.archive_files(output.getvalue(), check.GA4)

    def test_manifest_contract_and_module_cannot_be_substituted(self):
        original = self.files["tool.yaml"]
        mutations = [(b"version: 0.2.0", b"version: 0.3.0"),
                     (b"kind: mcp-server", b"kind: cli"),
                     (b"availability: setup-required", b"availability: not-verified"),
                     (b"GA4_CREDENTIAL", b"OTHER_CREDENTIAL"),
                     (b"scripts/ga4_mcp_server.mjs", b"scripts/other.mjs")]
        with tempfile.TemporaryDirectory() as temporary:
            path = self.package(Path(temporary) / "package") / "tool.yaml"
            check.inspect_pinned(self.verifier, path, self.manifest_sha, sample=check.GA4)
            with self.assertRaises(ValueError):
                check.inspect_pinned(self.verifier, path, self.manifest_sha, sample={**check.GA4, "module": "skills"})
            for before, after in mutations:
                with self.subTest(before=before):
                    self.assertIn(before, original)
                    altered = original.replace(before, after)
                    path.write_bytes(altered)
                    with self.assertRaises(ValueError):
                        check.inspect_pinned(self.verifier, path, check.sha256(altered), sample=check.GA4)

    def test_payload_permissions_inventory_and_complete_receipt_are_verified(self):
        for changes in [{"module": "skills"}, {"id": "analytics/another"}, {"version": "0.3.0"},
                        {"source": "local"}, {"runtime": "codex"}, {"artifact_sha256": "0" * 64},
                        {"tree_sha256": "0" * 64}, {"runtime_contract_sha256": "0" * 64},
                        {"installed_at": "invalid"}, {"unknown": True}]:
            with self.subTest(changes=changes), tempfile.TemporaryDirectory() as temporary:
                package = self.package(Path(temporary) / "package")
                self.verify(package)
                self.package(package, **changes)
                with self.assertRaises(ValueError):
                    self.verify(package)
        for mutation in ["bytes", "mode", "link", "extra", "empty-directory"]:
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as temporary:
                package = self.package(Path(temporary) / "package")
                if mutation == "bytes":
                    (package / "README.md").write_text("changed")
                elif mutation == "mode":
                    path = package / "scripts/ga4_mcp_server.mjs"
                    path.chmod(0o644 if self.modes["scripts/ga4_mcp_server.mjs"] == 0o755 else 0o755)
                elif mutation == "link":
                    (package / "README.md").unlink()
                    (package / "README.md").symlink_to(package / "TOOL.md")
                elif mutation == "extra":
                    (package / "extra").write_text("unexpected")
                else:
                    (package / "extra").mkdir()
                with self.assertRaises(ValueError):
                    self.verify(package)

    def test_actual_retained_server_starts_with_no_parent_credentials(self):
        with tempfile.TemporaryDirectory() as temporary, patch.dict(os.environ, {
                "GA4_CREDENTIAL": "must-not-be-inherited", "GOOGLE_APPLICATION_CREDENTIALS": "/must/not/use",
                "NODE_OPTIONS": "--this-would-break-node-if-inherited"}):
            probe = b"if (['GA4_CREDENTIAL','GOOGLE_APPLICATION_CREDENTIALS','NODE_OPTIONS','HOME'].some(k=>process.env[k])) throw new Error('inherited runtime environment');\n"
            result = self.startup(Path(temporary), {**self.files, "scripts/ga4_mcp_server.mjs":
                                                  probe + self.files["scripts/ga4_mcp_server.mjs"].split(b"\n", 1)[1]})
            self.assertEqual(result["network_attempts"], 0)
            self.assertFalse(result["credentials_supplied"])
            self.assertFalse(result["provider_calls_verified"])
            for key in ["setup_guidance", "healthcheck", "mcp_initialize", "mcp_tools_list"]:
                self.assertEqual(result[key], "passed")

    def test_registration_rejects_other_commands_paths_or_credential_injection(self):
        for guide in [self.guide.replace('"command": "node"', '"command": "sh"'),
                      self.guide.replace("scripts/ga4_mcp_server.mjs", "scripts/other.mjs"),
                      self.guide.replace('"type": "stdio"', '"env": {"GA4_CREDENTIAL": "value"}, "type": "stdio"'),
                      self.guide + '\n```json\n{}\n```']:
            with self.subTest(guide=guide[-80:]), self.assertRaises(ValueError):
                check.ga4_startup.registration(guide, Path("/installed"))

    def test_missing_guidance_rejects_before_executing_help_or_server(self):
        with patch.object(check.ga4_startup.subprocess, "run") as run, self.assertRaises(ValueError):
            check.ga4_startup.verify(self.binary, b"read README\n", Path("/package"), Path("/work"))
        run.assert_not_called()

    def test_constant_health_success_cannot_fake_mcp_startup(self):
        fake = b'console.log(JSON.stringify({status:"ok",server:"ga4-mcp-connector",version:"0.2.0",transport:"stdio"}));\n'
        with tempfile.TemporaryDirectory() as temporary, self.assertRaises(ValueError):
            self.startup(Path(temporary), {**self.files, "scripts/ga4_mcp_server.mjs": fake})

    def test_wrong_protocol_tool_or_input_contract_cannot_pass_startup(self):
        original = self.files["scripts/ga4_mcp_server.mjs"]
        for before, after in [(b'protocolVersion: supported.has(requested) ? requested : "2025-06-18"', b'protocolVersion: "2024-11-05"'),
                              (b'name: "ga4_run_report"', b'name: "other_tool"'),
                              (b'required: ["property_id", "start_date", "end_date", "metrics"]', b'required: []')]:
            self.assertIn(before, original)
            with self.subTest(before=before), tempfile.TemporaryDirectory() as temporary, self.assertRaises(ValueError):
                self.startup(Path(temporary), {**self.files, "scripts/ga4_mcp_server.mjs": original.replace(before, after)})

    def test_malformed_or_extra_stdout_fails_instead_of_becoming_evidence(self):
        node = shutil.which("node")
        for output in ["not json\n", '{}\nextra\n']:
            with self.subTest(output=output), tempfile.TemporaryDirectory() as temporary, self.assertRaises(ValueError):
                with check.ga4_startup.Process([node, "-e", f"process.stdout.write({json.dumps(output)})"], temporary) as process:
                    process.read()
                    process.finish()

    def test_caught_network_attempts_are_still_rejected(self):
        for attempt in [b"try { await fetch('https://must-not-be-contacted.invalid'); } catch {}\n",
                        b"import net from 'node:net'; try { net.connect(443, 'must-not-be-contacted.invalid'); } catch {}\n"]:
            with self.subTest(attempt=attempt), tempfile.TemporaryDirectory() as temporary, \
                    self.assertRaisesRegex(ValueError, "attempted network"):
                self.startup(Path(temporary), {**self.files, "scripts/ga4_mcp_server.mjs":
                                             attempt + self.files["scripts/ga4_mcp_server.mjs"].split(b"\n", 1)[1]})

    def test_process_deadline_and_output_budget_reap_children(self):
        node = shutil.which("node")
        for code, error in [("setInterval(()=>{},1000)", "timed out"),
                            ("process.stderr.write('x'.repeat(300000));setInterval(()=>{},1000)", "output exceeded")]:
            with self.subTest(error=error), tempfile.TemporaryDirectory() as temporary:
                with self.assertRaisesRegex(ValueError, error):
                    with check.ga4_startup.Process([node, "-e", code], temporary, timeout=1) as process:
                        process.read()
                self.assertIsNotNone(process.child.poll())

    def test_server_exit_before_initialization_closes_every_pipe(self):
        with tempfile.TemporaryDirectory() as temporary:
            with self.assertRaisesRegex(ValueError, "closed input"):
                with check.ga4_startup.Process([shutil.which("node"), "-e", "process.exit(0)"], temporary) as process:
                    process.child.wait(timeout=5)
                    process.send({"jsonrpc": "2.0", "id": 1, "method": "initialize"})
            self.assertIsNotNone(process.child.poll())
            self.assertTrue(all(stream.closed for stream in
                                [process.child.stdin, process.child.stdout, process.child.stderr]))

    def test_mcp_envelopes_cannot_substitute_request_or_error_for_success(self):
        for document in [{"jsonrpc": "2.0", "id": True, "result": {}},
                         {"jsonrpc": "2.0", "id": 2, "result": {}},
                         {"jsonrpc": "1.0", "id": 1, "result": {}},
                         {"jsonrpc": "2.0", "id": 1, "result": {}, "error": {}},
                         {"jsonrpc": "2.0", "id": 1, "result": []}]:
            with self.subTest(document=document), self.assertRaises(ValueError):
                check.ga4_startup.response(document, 1)

    def test_startup_mutation_fails_before_duplicate_or_repair(self):
        script = b"import fs from 'node:fs'; fs.appendFileSync(new URL('../README.md', import.meta.url), 'mutated');\n"
        files = {**self.files, "scripts/ga4_mcp_server.mjs": script + self.files["scripts/ga4_mcp_server.mjs"].split(b"\n", 1)[1]}
        real_run = subprocess.run
        installs = []
        with tempfile.TemporaryDirectory() as temporary:
            directory = Path(temporary)
            pinned = self.package(directory / "pinned", files)
            def install(args, **kwargs):
                if len(args) > 1 and args[0] == str(self.binary) and args[1] == "install":
                    installs.append(args)
                    self.package(directory / "generic/runtime/tools/analytics/ga4-mcp-connector", files)
                    return subprocess.CompletedProcess(args, 0, b"install.setup_guidance: skills-hub help setup\n", b"")
                return real_run(args, **kwargs)
            with patch.object(check.subprocess, "run", side_effect=install), \
                    self.assertRaisesRegex(ValueError, "startup mutated"):
                check.verify_runtime(self.binary, check.GA4["registry"], "generic", directory, files, self.artifact_sha,
                                     {}, self.verifier, pinned / "tool.yaml", self.manifest_sha, check.GA4, self.modes)
            self.assertEqual(len(installs), 1)

    def test_failed_release_prerequisite_stays_release_and_records_ga4_scope(self):
        with tempfile.TemporaryDirectory() as temporary:
            report = Path(temporary) / "report.json"
            with patch("sys.argv", ["check", "--sample", "ga4", "--cli-release", "", "--report", str(report)]), \
                    contextlib.redirect_stdout(io.StringIO()), patch.object(check, "download") as download:
                self.assertEqual(check.main(), 1)
            result = json.loads(report.read_text())
            self.assertEqual(result["client_kind"], "release")
            self.assertEqual(result["module"], "tools")
            self.assertEqual(result["registry_url"], check.GA4["registry"])
            self.assertEqual(result["failure_stage"], "client")
            download.assert_not_called()


if __name__ == "__main__":
    unittest.main()
