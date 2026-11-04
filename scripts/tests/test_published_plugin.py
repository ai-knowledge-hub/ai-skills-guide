"""Falsify incomplete or misleading plugin delivery using frozen-client fixtures."""

import contextlib
import copy
import hashlib
import importlib.util
import io
import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch


ROOT = Path(__file__).resolve().parents[2]
FIXTURES = ROOT / "scripts/tests/fixtures/code-maintenance-0.2.0"
spec = importlib.util.spec_from_file_location("plugin_delivery_check", ROOT / "scripts/check-published-install.py")
check = importlib.util.module_from_spec(spec)
spec.loader.exec_module(check)
plugin = check.plugin_check
IDS = ["engineering/code-change-verification", "engineering/coverage-gap-reporter", "engineering/implementation-strategy",
       "engineering/pr-review-and-draft", "engineering/test-gap-analyzer"]
ARTIFACT = "7d170bcd18bb15c58eacd99fc8a0def78498ca795e62eae9fc951301b308f93a"


def fixture_hash(root):
    digest = hashlib.sha256()
    for path in sorted(root.rglob("*")):
        name = path.relative_to(root).as_posix()
        if name == ".skills-hub-install.json":
            continue
        kind = "dir" if path.is_dir() else "file"
        mode = "755" if path.is_dir() or path.stat().st_mode & 0o111 else "644"
        digest.update(f"{kind}\0{name}\0{mode}\0".encode())
        if path.is_file():
            digest.update(path.read_bytes())
        digest.update(b"\0")
    return digest.hexdigest()


class PublishedPluginTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.build = tempfile.TemporaryDirectory(prefix="plugin-delivery-tests-")
        cls.verifier = Path(cls.build.name) / "verifier"
        subprocess.run(["go", "build", "-o", str(cls.verifier), "./cmd/published-install-verifier"], cwd=ROOT,
                       env={**os.environ, "CGO_ENABLED": "0", "GOFLAGS": "", "GOWORK": "off"}, check=True)
        archive = (ROOT / "releases/plugins/engineering/code-maintenance-plugin/0.2.0/package.tar.gz").read_bytes()
        if hashlib.sha256(archive).hexdigest() != ARTIFACT:
            raise AssertionError("retained plugin archive changed")
        cls.modes = {}
        cls.files = check.archive_files(archive, check.CODE_MAINTENANCE, cls.modes)
        cls.pinned_root = Path(cls.build.name) / "pinned"
        cls.write_files(cls.pinned_root, cls.files, cls.modes)
        cls.manifest_sha = hashlib.sha256(cls.files["plugin.yaml"]).hexdigest()
        cls.pinned = check.inspect_pinned(cls.verifier, cls.pinned_root / "plugin.yaml", cls.manifest_sha,
                                         sample=check.CODE_MAINTENANCE)
        cls.receipts = json.loads((FIXTURES / "receipts.json").read_text())
        if sorted(cls.receipts["skills"]) != IDS:
            raise AssertionError("frozen fixture omits a required skill")

    @classmethod
    def tearDownClass(cls):
        cls.build.cleanup()

    @staticmethod
    def write_files(root, files, modes):
        root.mkdir(parents=True, exist_ok=True)
        for name, data in files.items():
            path = root / name
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(data)
        for name, mode in modes.items():
            (root / name).chmod(mode)

    def fixture(self, runtime_root, runtime="generic", files=None):
        files = self.files if files is None else files
        root = runtime_root / "plugins/engineering/code-maintenance-plugin"
        self.write_files(root, files, self.modes)
        for id in IDS:
            bundled = root / "bundled/skills" / id
            shutil.copytree(bundled, root / "skills" / id.split("/")[-1])
            sibling = runtime_root / "skills" / id
            shutil.copytree(bundled, sibling)
            record = copy.deepcopy(self.receipts["skills"][id])
            self.assertEqual(record["tree_sha256"], fixture_hash(sibling))
            record["runtime"] = runtime
            for path in [bundled / plugin.RECEIPT, sibling / plugin.RECEIPT]:
                path.write_text(json.dumps(record, indent=2) + "\n")
                path.chmod(0o644)
        (root / ".runtime").mkdir()
        shutil.copyfile(FIXTURES / f"{runtime}-runtime.json", root / f".runtime/{runtime}.json")
        if runtime == "codex":
            shutil.copyfile(FIXTURES / "codex-native.json", root / "plugin.json")
        elif runtime == "claude":
            (root / ".claude-plugin").mkdir()
            shutil.copyfile(FIXTURES / "claude-native.json", root / ".claude-plugin/plugin.json")
        record = copy.deepcopy(self.receipts["root"])
        record["runtime"] = runtime
        record["tree_sha256"] = fixture_hash(root)
        (root / plugin.RECEIPT).write_text(json.dumps(record, indent=2) + "\n")
        (root / plugin.RECEIPT).chmod(0o644)
        return root

    def inspector(self, runtime):
        return lambda package: check.inspect_pinned(self.verifier, self.pinned_root / "plugin.yaml", self.manifest_sha,
                                                    package, runtime, ARTIFACT, check.CODE_MAINTENANCE)

    def verify(self, runtime_root, runtime="generic", files=None):
        return plugin.verify_install(runtime_root, runtime, self.files if files is None else files, self.modes,
                                     ARTIFACT, self.pinned, self.inspector(runtime))

    def rehash_root(self, root, **changes):
        path = root / plugin.RECEIPT
        record = json.loads(path.read_text())
        record.update(changes)
        record["tree_sha256"] = fixture_hash(root)
        path.write_text(json.dumps(record) + "\n")

    def test_frozen_released_client_layouts_pass_all_three_adapters(self):
        plugin.verify_archive(self.files, self.modes)
        for runtime in ["codex", "claude", "generic"]:
            with self.subTest(runtime=runtime), tempfile.TemporaryDirectory() as temporary:
                root = Path(temporary) / "runtime"
                self.fixture(root, runtime)
                result = self.verify(root, runtime)
                self.assertTrue(result["receipt_verified"] and result["closure_verified"])
                self.assertEqual(result["dependency_count"], 5)
                self.assertEqual(result["hook_enforcement"], "advisory")

    def test_lock_and_bundle_cannot_coherently_omit_one_required_skill(self):
        missing = IDS[0]
        files = {k: v for k, v in self.files.items() if not k.startswith(f"bundled/skills/{missing}/")}
        lock = json.loads(files["dependencies.lock.json"])
        lock["components"] = [m for m in lock["components"] if m["id"] != missing]
        files["dependencies.lock.json"] = json.dumps(lock).encode()
        files["plugin.yaml"] = files["plugin.yaml"].replace(f"    - {missing}\n".encode(), b"")
        files["checksums.txt"] = "".join(f"{hashlib.sha256(v).hexdigest()}  {k}\n" for k, v in sorted(files.items())
                                           if k != "checksums.txt").encode()
        modes = {k: self.modes[k] for k in plugin.inventory(files)}
        with self.assertRaisesRegex(ValueError, "all five"):
            plugin.verify_archive(files, modes)

    def test_component_path_digest_edges_and_version_are_pinned_independently(self):
        for change in [{"manifest_path": "bundled/skills/other/skill.yaml"}, {"version": "0.2.0"},
                       {"module": "agents"}, {"content_sha256": "0" * 64}, {"dependencies": ["skills:other/component"]}]:
            files = dict(self.files)
            lock = json.loads(files["dependencies.lock.json"])
            lock["components"][0].update(change)
            files["dependencies.lock.json"] = json.dumps(lock).encode()
            with self.subTest(change=change), self.assertRaisesRegex(ValueError, "lock component"):
                plugin.verify_archive(files, self.modes)

    def test_checksums_cannot_omit_or_duplicate_a_payload_member(self):
        rows = self.files["checksums.txt"].splitlines(keepends=True)
        for data in [b"".join(rows[1:]), b"".join(rows + rows[:1]), b"0" * 64 + b"  README.md\n"]:
            with self.subTest(data=data[:20]), self.assertRaises(ValueError):
                plugin.verify_archive({**self.files, "checksums.txt": data}, self.modes)

    def test_undeclared_raw_receipts_empty_directories_and_bundled_components_fail(self):
        for name in [plugin.RECEIPT, "bundled/skills/other/component/skill.yaml"]:
            with self.subTest(name=name), self.assertRaises(ValueError):
                plugin.verify_archive({**self.files, name: b"untrusted"}, {**self.modes, name: 0o644})
        with self.assertRaisesRegex(ValueError, "empty directories"):
            plugin.verify_archive(self.files, {**self.modes, "empty": 0o755})

    def test_receipt_defined_closure_cannot_hide_a_coordinated_missing_skill(self):
        with tempfile.TemporaryDirectory() as temporary:
            runtime = Path(temporary) / "runtime"
            root = self.fixture(runtime)
            missing = IDS[0]
            shutil.rmtree(root / "bundled/skills" / missing)
            shutil.rmtree(root / "skills" / missing.split("/")[-1])
            shutil.rmtree(runtime / "skills" / missing)
            contract = root / ".runtime/generic.json"
            doc = json.loads(contract.read_text())
            doc["registrations"]["skills"] = [x for x in doc["registrations"]["skills"] if x["id"] != missing]
            contract.write_text(json.dumps(doc))
            record = json.loads((root / plugin.RECEIPT).read_text())
            self.rehash_root(root, closure=[x for x in record["closure"] if x["id"] != missing])
            with self.assertRaises(ValueError):
                self.verify(runtime)

    def test_sibling_dependency_tamper_fails_even_when_plugin_tree_is_unchanged(self):
        for mutation in ["bytes", "mode", "symlink", "hardlink", "extra-directory"]:
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as temporary:
                runtime = Path(temporary) / "runtime"
                root = self.fixture(runtime)
                before = fixture_hash(root)
                member = runtime / "skills" / IDS[0]
                path = member / "SKILL.md"
                if mutation == "bytes": path.write_text("substituted")
                elif mutation == "mode": path.chmod(0o755)
                elif mutation == "symlink":
                    path.unlink();path.symlink_to(member / "README.md")
                elif mutation == "hardlink": (Path(temporary) / "outside").hardlink_to(path)
                else: (member / "extra").mkdir()
                self.assertEqual(fixture_hash(root), before)
                with self.assertRaises(ValueError): self.verify(runtime)

    def test_bundled_receipts_cannot_substitute_parent_source_version_or_contract(self):
        for change in [{"source": "local"}, {"parent_artifact_sha256": "0" * 64}, {"version": "0.2.0"},
                       {"runtime": "codex"}, {"runtime_contract_sha256": "0" * 64}, {"installed_at": "invalid"},
                       {"artifact_sha256": ARTIFACT}, {"unexpected": True}]:
            with self.subTest(change=change), tempfile.TemporaryDirectory() as temporary:
                runtime = Path(temporary) / "runtime"
                root = self.fixture(runtime)
                paths = [root / "bundled/skills" / IDS[0] / plugin.RECEIPT,
                         runtime / "skills" / IDS[0] / plugin.RECEIPT]
                record = json.loads(paths[0].read_text());record.update(change)
                for path in paths: path.write_text(json.dumps(record))
                self.rehash_root(root)
                with self.assertRaises(ValueError): self.verify(runtime)
                with self.assertRaises(ValueError): self.inspector("generic")(root)

    def test_root_receipts_cannot_omit_duplicate_or_substitute_closure_members(self):
        original = self.receipts["root"]["closure"]
        for closure in [[], original[:-1], original[:-1] + original[:1],
                        [{**original[0], "version": "0.2.0"}, *original[1:]],
                        [{**original[0], "runtime_contract_sha256": "0" * 64}, *original[1:]]]:
            with self.subTest(closure=closure[:1]), tempfile.TemporaryDirectory() as temporary:
                runtime = Path(temporary) / "runtime"
                root = self.fixture(runtime)
                self.rehash_root(root, closure=closure)
                with self.assertRaises(ValueError): self.verify(runtime)
                with self.assertRaises(ValueError): self.inspector("generic")(root)

    def test_registration_and_advisory_hook_cannot_be_downgraded_or_promoted(self):
        for runtime_name in ["codex", "claude", "generic"]:
            for mutation in ["missing-skill", "wrong-runtime", "native-hook", "extra-authority"]:
                with self.subTest(runtime=runtime_name, mutation=mutation), tempfile.TemporaryDirectory() as temporary:
                    runtime = Path(temporary) / "runtime"
                    root = self.fixture(runtime, runtime_name)
                    path = root / f".runtime/{runtime_name}.json"
                    document = json.loads(path.read_text())
                    if mutation == "missing-skill": document["registrations"]["skills"].pop()
                    elif mutation == "wrong-runtime": document["runtime"] = "other"
                    elif mutation == "native-hook": document["registrations"]["hooks"][0]["enforcement"] = "native"
                    else: document["merge_authorized"] = True
                    path.write_text(json.dumps(document));self.rehash_root(root)
                    with self.assertRaises(ValueError): self.verify(runtime, runtime_name)

    def test_native_metadata_extra_discovery_and_materialized_skill_writes_fail(self):
        for mutation in ["codex-metadata", "claude-hook", "extra-discovery", "native-skill", "outside-package"]:
            name = "claude" if mutation == "claude-hook" else "codex"
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as temporary:
                runtime = Path(temporary) / "runtime"
                root = self.fixture(runtime, name)
                if mutation == "codex-metadata": (root / "plugin.json").write_text('{"name":"minimal"}')
                elif mutation == "claude-hook":
                    path = root / ".claude-plugin/plugin.json"
                    data = json.loads(path.read_text());data["hooks"] = "./hooks/verification-before-complete.md"
                    path.write_text(json.dumps(data))
                elif mutation == "extra-discovery": (root / ".mcp.json").write_text("{}")
                elif mutation == "native-skill": (root / "skills/code-change-verification/SKILL.md").write_text("substituted")
                else: (runtime / "skills/engineering/other-package").mkdir()
                self.rehash_root(root)
                with self.assertRaises(ValueError): self.verify(runtime, name)

    def test_actual_first_use_is_read_only_with_no_inherited_credentials(self):
        probe = b"if (['HOME','GH_TOKEN','NODE_OPTIONS','BQ_CREDENTIAL'].some(k=>process.env[k])) throw new Error('inherited environment');\n"
        with tempfile.TemporaryDirectory() as temporary, patch.dict(os.environ, {
                "GH_TOKEN": "not-inherited", "BQ_CREDENTIAL": "not-inherited", "NODE_OPTIONS": "--invalid-inherited-option"}):
            runtime = Path(temporary) / "runtime"
            files = {**self.files, "scripts/first_use.mjs": probe + self.files["scripts/first_use.mjs"].split(b"\n", 1)[1]}
            root = self.fixture(runtime, files=files)
            work = Path(temporary) / "work";work.mkdir()
            before = plugin.closure_snapshot(runtime, check.filesystem_snapshot)
            result = plugin.first_use(root, files, work, check.mcp_startup)
            self.assertEqual(result["probes"], 2)
            self.assertFalse(result["merge_authorized"])
            self.assertFalse(result["verification_command_executed"])
            self.assertEqual(result["network_attempts"], 0)
            self.assertEqual(plugin.closure_snapshot(runtime, check.filesystem_snapshot), before)

    def test_constant_output_extra_stdout_and_merge_authority_cannot_pass_first_use(self):
        sample = json.loads(self.files["examples/first-use-input.json"])
        constant = {"schema_version": "local-plugin.first-use/v1", "plugin": "engineering/code-maintenance-plugin",
                    "plan": ["inspect affected boundary", "implement the smallest change", sample["verification_command"], "report residual risk"],
                    "change": sample["change"], "risk": sample["risk"], "merge_authorized": False}
        for script in [f"console.log(JSON.stringify({json.dumps(constant)}));".encode(),
                       self.files["scripts/first_use.mjs"].replace(b"merge_authorized: false", b"merge_authorized: true"),
                       self.files["scripts/first_use.mjs"].replace(b"merge_authorized: false", b"merge_authorized: 0"),
                       self.files["scripts/first_use.mjs"] + b"console.log('extra');\n"]:
            with self.subTest(script=script[:35]), tempfile.TemporaryDirectory() as temporary:
                files = {**self.files, "scripts/first_use.mjs": script}
                root = self.fixture(Path(temporary) / "runtime", files=files)
                work = Path(temporary) / "work";work.mkdir()
                with self.assertRaises(ValueError): plugin.first_use(root, files, work, check.mcp_startup)

    def test_caught_network_and_executed_verification_command_fail_first_use(self):
        script = self.files["scripts/first_use.mjs"].split(b"\n", 1)[1]
        network = b"try { await fetch('https://must-not-be-contacted.invalid'); } catch {}\n" + script
        execution = script.replace(b"process.stdout.write", b"if (input.verification_command.startsWith('touch ')) (await import('node:child_process')).execSync(input.verification_command);\nprocess.stdout.write")
        for data, message in [(network, "attempted network"), (execution, "executed the proposed")]:
            with self.subTest(message=message), tempfile.TemporaryDirectory() as temporary:
                files = {**self.files, "scripts/first_use.mjs": data}
                root = self.fixture(Path(temporary) / "runtime", files=files)
                work = Path(temporary) / "work";work.mkdir()
                with self.assertRaisesRegex(ValueError, message): plugin.first_use(root, files, work, check.mcp_startup)

    def test_failed_release_prerequisite_preserves_plugin_scope(self):
        with tempfile.TemporaryDirectory() as temporary:
            report = Path(temporary) / "report.json"
            with patch("sys.argv", ["check", "--sample", "code-maintenance", "--cli-release", "", "--report", str(report)]), \
                    contextlib.redirect_stdout(io.StringIO()), patch.object(check, "download") as download:
                self.assertEqual(check.main(), 1)
            record = json.loads(report.read_text())
            self.assertEqual(record["module"], "plugins")
            self.assertEqual(record["client_kind"], "release")
            self.assertEqual(record["package"], "engineering/code-maintenance-plugin@0.2.0")
            download.assert_not_called()

    def test_registry_cannot_change_artifact_under_the_pinned_plugin_version(self):
        pinned = {"version": "0.2.0", "sha256": ARTIFACT, "manifest_sha256": self.manifest_sha}
        entry = {"id": "engineering/code-maintenance-plugin", "latest": "99.0.0", "versions": [pinned, {"version": "99.0.0"}]}
        index = {"registry_version": "1.3", "skills": [entry]}
        self.assertEqual(check.selected_release(index, check.CODE_MAINTENANCE), pinned)
        pinned["sha256"] = "0" * 64
        with self.assertRaisesRegex(ValueError, "reviewed release"):
            check.selected_release(index, check.CODE_MAINTENANCE)

    def test_failed_transitions_cannot_hide_mutation_or_incomplete_offline_activation(self):
        real_run = subprocess.run
        for scenario, expected_calls in [("first-use-mutates", 1), ("duplicate-mutates", 2), ("wrong-error", 2),
                                          ("force-mutates", 3), ("offline-incomplete", 4)]:
            with self.subTest(scenario=scenario), tempfile.TemporaryDirectory() as temporary:
                directory = Path(temporary)
                files = self.files
                if scenario == "first-use-mutates":
                    prefix = b"import {appendFileSync as disturb} from 'node:fs'; disturb(new URL('../README.md',import.meta.url),'mutated');\n"
                    files = {**files, "scripts/first_use.mjs": prefix + files["scripts/first_use.mjs"].split(b"\n", 1)[1]}
                calls = []
                def install(args, **kwargs):
                    if args[0] != "fake-cli": return real_run(args, **kwargs)
                    calls.append(args)
                    target = Path(args[args.index("--target") + 1]).parent
                    if len(calls) == 1 or target.name == "offline-runtime":
                        self.fixture(target, files=files)
                        if target.name == "offline-runtime" and scenario == "offline-incomplete":
                            shutil.rmtree(target / "skills" / IDS[0])
                        return subprocess.CompletedProcess(args, 0, b"", b"")
                    if len(calls) == 2:
                        if scenario == "duplicate-mutates":
                            (target / "skills" / IDS[0] / "SKILL.md").chmod(0o755)
                        return subprocess.CompletedProcess(args, 1, b"", b"unrelated failure" if scenario == "wrong-error" else b"destination already exists")
                    if scenario == "force-mutates": (target / "skills" / IDS[0] / "README.md").write_text("mutated")
                    return subprocess.CompletedProcess(args, 1, b"", b"in-place plugin closure upgrade is not supported")
                with patch.object(check.subprocess, "run", side_effect=install), self.assertRaises(ValueError):
                    plugin.verify_runtime(Path("fake-cli"), check.CODE_MAINTENANCE["registry"], "generic", directory, files,
                                          self.modes, ARTIFACT, {}, self.inspector("generic"), check.filesystem_snapshot, check.mcp_startup)
                self.assertEqual(len(calls), expected_calls)


if __name__ == "__main__":
    unittest.main()
