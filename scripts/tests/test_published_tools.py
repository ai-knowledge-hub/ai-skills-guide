"""Independent pinned cases for published BigQuery and Meta Ads MCP packages."""

import contextlib
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch


ROOT = Path(__file__).resolve().parents[2]
spec = importlib.util.spec_from_file_location("published_tools_check", ROOT / "scripts/check-published-install.py")
check = importlib.util.module_from_spec(spec)
spec.loader.exec_module(check)

# Contracts observed from the frozen v0.4.0-alpha.1 client's public installs,
# rather than calculated by the current Go verifier under test.
CASES = {
    "bigquery": {"id": "warehouse/bigquery-mcp-query-runner", "server": "bigquery-mcp-query-runner",
                 "key": "bigquery", "script": "scripts/bigquery_mcp_server.mjs", "binding": "BQ_CREDENTIAL",
                 "tool": "bigquery_run_query", "contract": "ab890938dd2894a8510eca0e43003e8529f01413ac63a14f0851219641e3742c",
                 "archive": "0ea4c8301f2292f4790ae8d18344fffc9bc6e4aa8dbf82244574f21608745871",
                 "required": b'required: ["query", "project_id", "billing_project_id", "location"]'},
    "meta-ads": {"id": "ads/meta-ads-mcp-connector", "server": "meta-ads-mcp-connector",
                 "key": "meta-ads", "script": "scripts/meta_ads_mcp_server.mjs", "binding": "META_ADS_CREDENTIAL",
                 "tool": "meta_ads_read_insights", "contract": "4ae770ff0d2ae0b5e69a06b453716f902d6773787744590a9e2c81628d67ace4",
                 "archive": "3dfbe1eecdf4f6587fa041fd0e27ee56c6951a76b881bca94209182bd0c2c61e",
                 "required": b'required: ["ad_account_id", "level", "start_date", "end_date"]'},
}


class PublishedToolTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.build = tempfile.TemporaryDirectory(prefix="published-tools-tests-")
        cls.verifier, cls.binary = Path(cls.build.name) / "verifier", Path(cls.build.name) / "skills-hub"
        environment = {**os.environ, "CGO_ENABLED": "0", "GOFLAGS": "", "GOWORK": "off"}
        for output, package, tags in [(cls.verifier, "published-install-verifier", []),
                                      (cls.binary, "skills-hub", ["-tags", "installer_release"])]:
            subprocess.run(["go", "build", *tags, "-o", str(output), f"./cmd/{package}"],
                           cwd=ROOT, env=environment, check=True)
        cls.guide = subprocess.check_output([str(cls.binary), "help", "setup"], text=True)
        cls.payloads = {}
        for name, case in CASES.items():
            archive = (ROOT / f"releases/tools-mcp/{case['id']}/0.2.0/package.tar.gz").read_bytes()
            if hashlib.sha256(archive).hexdigest() != case["archive"]:
                raise AssertionError("retained test archive changed")
            modes = {}
            files = check.archive_files(archive, check.SAMPLES[name], modes)
            cls.payloads[name] = (files, modes)

    @classmethod
    def tearDownClass(cls):
        cls.build.cleanup()

    def package(self, root, name, files=None, **changes):
        original, modes = self.payloads[name]
        files = original if files is None else files
        root.mkdir(parents=True, exist_ok=True)
        for path, data in files.items():
            destination = root / path
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_bytes(data)
        for path, mode in modes.items():
            (root / path).chmod(mode)
        digest = hashlib.sha256()
        for path in sorted(root.rglob("*")):
            if path.name == check.RECEIPT:
                continue
            kind = "dir" if path.is_dir() else "file"
            digest.update(f"{kind}\0{path.relative_to(root).as_posix()}\0{path.stat().st_mode & 0o777:o}\0".encode())
            if path.is_file():
                digest.update(path.read_bytes())
            digest.update(b"\0")
        receipt = {"schema_version": "skills-hub.install-receipt/v1", "source": "remote", "module": "tools",
                   "id": CASES[name]["id"], "version": "0.2.0", "runtime": "generic",
                   "artifact_sha256": CASES[name]["archive"], "runtime_contract_sha256": CASES[name]["contract"],
                   "tree_sha256": digest.hexdigest(), "installed_at": "2026-11-04T00:00:00Z"}
        receipt.update(changes)
        (root / check.RECEIPT).write_text(json.dumps(receipt))
        return root

    def verify(self, root, name, files=None):
        original, modes = self.payloads[name]
        return check.verify_installed(root, original if files is None else files, "generic", CASES[name]["archive"],
                                      self.verifier, root / "tool.yaml", hashlib.sha256(original["tool.yaml"]).hexdigest(),
                                      check.SAMPLES[name], modes)

    def startup(self, root, name, files=None):
        package = self.package(root / "package", name, files)
        work = root / "outside-package"
        work.mkdir()
        return check.mcp_startup.verify(self.binary, b"install.setup_guidance: skills-hub help setup\n", package, work, name)

    def test_actual_retained_servers_launch_without_inherited_credentials_or_policy(self):
        probe = b"if (['BQ_CREDENTIAL','BQ_POLICY','META_ADS_CREDENTIAL','META_ADS_POLICY','GOOGLE_APPLICATION_CREDENTIALS','NODE_OPTIONS','HOME'].some(k=>process.env[k])) throw new Error('inherited runtime environment');\n"
        for name, case in CASES.items():
            files, _ = self.payloads[name]
            isolated = {**files, case["script"]: probe + files[case["script"]].split(b"\n", 1)[1]}
            with self.subTest(name=name), tempfile.TemporaryDirectory() as temporary, patch.dict(os.environ, {
                    "BQ_CREDENTIAL": "not-inherited", "BQ_POLICY": "not-inherited", "META_ADS_CREDENTIAL": "not-inherited",
                    "META_ADS_POLICY": "not-inherited", "GOOGLE_APPLICATION_CREDENTIALS": "/not/inherited",
                    "NODE_OPTIONS": "--this-would-break-node"}):
                result = self.startup(Path(temporary), name, isolated)
                self.assertEqual(result["server"], case["server"])
                self.assertEqual(result["network_attempts"], 0)
                self.assertFalse(result["credentials_supplied"])
                self.assertFalse(result["provider_calls_verified"])
                for key in ["setup_guidance", "healthcheck", "mcp_initialize", "mcp_tools_list"]:
                    self.assertEqual(result[key], "passed")

    def test_packaged_registration_and_prose_guide_must_agree_with_each_pin(self):
        for name, case in CASES.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as temporary:
                root = self.package(Path(temporary) / "package", name)
                registration = check.mcp_startup.registration(self.guide, root, name)
                self.assertEqual(registration, {"type": "stdio", "command": "node",
                                               "args": ["--preserve-symlinks-main", str(root / case["script"])]})
                for before, after in [(case["id"], "other/package"), (case["script"], "scripts/other.mjs"),
                                      (case["binding"], "OTHER_CREDENTIAL")]:
                    with self.subTest(before=before), self.assertRaises(ValueError):
                        check.mcp_startup.registration(self.guide.replace(before, after), root, name)
                original = json.loads((root / ".mcp.json").read_text())
                for change in [{"command": "sh"}, {"args": ["/outside/other.mjs"]}, {"cwd": "/outside"},
                               {"env": {case["binding"]: "value"}}]:
                    settings = json.loads(json.dumps(original))
                    settings["mcpServers"][case["key"]].update(change)
                    (root / ".mcp.json").write_text(json.dumps(settings))
                    with self.subTest(change=change), self.assertRaises(ValueError):
                        check.mcp_startup.registration(self.guide, root, name)

    def test_valid_other_tool_cannot_substitute_for_selected_sample(self):
        for name in CASES:
            other = next(key for key in CASES if key != name)
            with self.subTest(name=name), tempfile.TemporaryDirectory() as temporary:
                root = self.package(Path(temporary) / "package", name)
                self.verify(root, name)
                with self.assertRaises(ValueError):
                    self.verify(root, other)
                with self.assertRaises(ValueError):
                    check.mcp_startup.registration(self.guide, root, other)

    def test_receipts_cannot_substitute_module_subject_runtime_source_or_integrity(self):
        for name in CASES:
            for changes in [{"module": "skills"}, {"id": "analytics/ga4-mcp-connector"}, {"version": "0.3.0"},
                            {"runtime": "claude"}, {"source": "local"}, {"artifact_sha256": "0" * 64},
                            {"runtime_contract_sha256": "0" * 64}, {"tree_sha256": "0" * 64},
                            {"installed_at": "invalid"}, {"unexpected": True}]:
                with self.subTest(name=name, changes=changes), tempfile.TemporaryDirectory() as temporary:
                    root = self.package(Path(temporary) / "package", name)
                    self.verify(root, name)
                    self.package(root, name, **changes)
                    with self.assertRaises(ValueError):
                        self.verify(root, name)

    def test_payload_bytes_permissions_and_extra_directories_are_independent_oracles(self):
        for name, case in CASES.items():
            for mutation in ["bytes", "permissions", "extra-directory", "link"]:
                with self.subTest(name=name, mutation=mutation), tempfile.TemporaryDirectory() as temporary:
                    root = self.package(Path(temporary) / "package", name)
                    if mutation == "bytes":
                        (root / case["script"]).write_text("changed")
                    elif mutation == "permissions":
                        path = root / case["script"]
                        path.chmod(0o644 if path.stat().st_mode & 0o111 else 0o755)
                    elif mutation == "extra-directory":
                        (root / "unexpected").mkdir()
                    else:
                        (root / "README.md").unlink()
                        (root / "README.md").symlink_to(root / "TOOL.md")
                    with self.assertRaises(ValueError):
                        self.verify(root, name)

    def test_historical_manifest_contract_cannot_be_reclassified_or_rebound(self):
        for name, case in CASES.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as temporary:
                root = self.package(Path(temporary) / "package", name)
                path = root / "tool.yaml"
                original = path.read_bytes()
                for before, after in [(b"version: 0.2.0", b"version: 0.3.0"),
                                      (b"availability: setup-required", b"availability: usable-now"),
                                      (b"kind: mcp-server", b"kind: cli"),
                                      (case["script"].encode(), b"scripts/other.mjs"),
                                      (case["binding"].encode(), b"OTHER_CREDENTIAL")]:
                    self.assertIn(before, original)
                    altered = original.replace(before, after)
                    path.write_bytes(altered)
                    with self.subTest(before=before), self.assertRaises(ValueError):
                        check.inspect_pinned(self.verifier, path, hashlib.sha256(altered).hexdigest(), sample=check.SAMPLES[name])

    def test_wrong_mcp_identity_tool_or_required_fields_fail_real_process_startup(self):
        for name, case in CASES.items():
            files, _ = self.payloads[name]
            original = files[case["script"]]
            for before, after in [(f'serverInfo: {{ name: "{case["server"]}"'.encode(), b'serverInfo: { name: "other-server"'),
                                  (f'name: "{case["tool"]}"'.encode(), b'name: "other_tool"'),
                                  (case["required"], b'required: []'),
                                  (case["required"], case["required"][:-1] + b', "unexpected_credential"]')]:
                self.assertIn(before, original)
                with self.subTest(name=name, before=before), tempfile.TemporaryDirectory() as temporary, self.assertRaises(ValueError):
                    self.startup(Path(temporary), name, {**files, case["script"]: original.replace(before, after)})

    def test_provider_specific_input_bounds_are_checked(self):
        for name, before, after in [("bigquery", b"maxLength: MAX_SQL_BYTES", b"maxLength: 1"),
                                    ("meta-ads", b"level: { enum: LEVELS }", b'level: { enum: ["account"] }')]:
            files, _ = self.payloads[name]
            script = CASES[name]["script"]
            self.assertIn(before, files[script])
            with self.subTest(name=name), tempfile.TemporaryDirectory() as temporary, self.assertRaises(ValueError):
                self.startup(Path(temporary), name, {**files, script: files[script].replace(before, after)})

    def test_caught_network_attempts_fail_for_both_servers(self):
        for name, case in CASES.items():
            files, _ = self.payloads[name]
            for attempt in [b"try { await fetch('https://must-not-be-contacted.invalid'); } catch {}\n",
                            b"import net from 'node:net'; try { net.connect(443,'must-not-be-contacted.invalid'); } catch {}\n"]:
                with self.subTest(name=name, attempt=attempt), tempfile.TemporaryDirectory() as temporary, \
                        self.assertRaisesRegex(ValueError, "attempted network"):
                    self.startup(Path(temporary), name, {**files, case["script"]:
                                 attempt + files[case["script"]].split(b"\n", 1)[1]})

    def test_startup_mutation_stops_before_duplicate_or_offline_repair(self):
        real_run = subprocess.run
        for name, case in CASES.items():
            original, modes = self.payloads[name]
            prefix = b"import {appendFileSync as disturb} from 'node:fs'; disturb(new URL('../README.md',import.meta.url),'mutated');\n"
            files = {**original, case["script"]: prefix + original[case["script"]].split(b"\n", 1)[1]}
            with self.subTest(name=name), tempfile.TemporaryDirectory() as temporary:
                directory = Path(temporary)
                pinned = self.package(directory / "pinned", name, files)
                calls = []
                def install(args, **kwargs):
                    if len(args) > 1 and args[:2] == [str(self.binary), "install"]:
                        calls.append(args)
                        self.package(directory / "generic/runtime/tools" / case["id"], name, files)
                        return subprocess.CompletedProcess(args, 0, b"install.setup_guidance: skills-hub help setup\n", b"")
                    return real_run(args, **kwargs)
                with patch.object(check.subprocess, "run", side_effect=install), self.assertRaisesRegex(ValueError, "startup mutated"):
                    check.verify_runtime(self.binary, check.SAMPLES[name]["registry"], "generic", directory, files, case["archive"],
                                         {}, self.verifier, pinned / "tool.yaml", hashlib.sha256(original["tool.yaml"]).hexdigest(),
                                         check.SAMPLES[name], modes)
                self.assertEqual(len(calls), 1)

    def test_selection_cannot_follow_latest_or_reuse_the_other_tool(self):
        for name, case in CASES.items():
            pinned = {"version": "0.2.0", "sha256": case["archive"], "manifest_sha256": "a" * 64}
            entry = {"id": case["id"], "latest": "99.0.0", "runtimes": [], "usability": {"availability": "template-only"},
                     "versions": [pinned, {"version": "99.0.0"}]}
            index = {"registry_version": "1.3", "skills": [entry]}
            with self.subTest(name=name):
                self.assertEqual(check.selected_release(index, check.SAMPLES[name]), pinned)
                other = next(key for key in CASES if key != name)
                with self.assertRaises(ValueError):
                    check.selected_release(index, check.SAMPLES[other])
                for versions in [[{"version": "99.0.0"}], [pinned, pinned]]:
                    with self.subTest(versions=versions), self.assertRaises(ValueError):
                        check.selected_release({"registry_version": "1.3", "skills": [{**entry, "versions": versions}]}, check.SAMPLES[name])

    def test_failed_release_prerequisite_preserves_selected_tool_scope(self):
        for name, case in CASES.items():
            with self.subTest(name=name), tempfile.TemporaryDirectory() as temporary:
                report = Path(temporary) / "report.json"
                with patch("sys.argv", ["check", "--sample", name, "--cli-release", "", "--report", str(report)]), \
                        contextlib.redirect_stdout(io.StringIO()), patch.object(check, "download") as download:
                    self.assertEqual(check.main(), 1)
                result = json.loads(report.read_text())
                self.assertEqual(result["client_kind"], "release")
                self.assertEqual(result["module"], "tools")
                self.assertEqual(result["package"], case["id"] + "@0.2.0")
                self.assertEqual(result["failure_stage"], "client")
                download.assert_not_called()


if __name__ == "__main__":
    unittest.main()
