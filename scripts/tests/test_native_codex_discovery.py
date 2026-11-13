"""Falsify discovery evidence without network, client downloads or model turns."""

import importlib.util
import io
import contextlib
import json
import os
from pathlib import Path
import sys
import tarfile
import tempfile
import unittest
from unittest.mock import patch


sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
import native_codex as native

spec = importlib.util.spec_from_file_location("native_check", Path(__file__).parents[1] / "check-native-codex-discovery.py")
check = importlib.util.module_from_spec(spec)
spec.loader.exec_module(check)


class DiscoveryEvidenceTests(unittest.TestCase):
    def setUp(self):
        self.repo = Path("/disposable/repository")
        self.instruction = Path("/disposable/home/.codex/skills/engineering/implementation-strategy/SKILL.md")
        self.skill = {"name": "implementation-strategy", "path": str(self.instruction),
                      "scope": "user", "enabled": True, "description": "Plan a change before editing."}

    def result(self, skills=None, errors=None):
        return {"data": [{"cwd": str(self.repo), "skills": skills or [], "errors": errors or []}]}

    def test_positive_result_requires_exact_identity_path_scope_enabled_and_no_errors(self):
        valid = self.result([self.skill])
        native.assert_discovery(valid, self.repo, self.instruction, "installed")
        for mutation in [{"name": "another-valid-skill"}, {"path": "/another/SKILL.md"},
                         {"scope": "repo"}, {"enabled": False}, {"enabled": "true"}, {"description": ""}]:
            with self.subTest(mutation=mutation), self.assertRaises(ValueError):
                native.assert_discovery(self.result([{**self.skill, **mutation}]), self.repo, self.instruction, "installed")
        for result in [self.result(), self.result([self.skill, self.skill]),
                       self.result([self.skill, {**self.skill, "name": "unrelated"}]),
                       self.result([self.skill], [{"path": "/other", "message": "error"}]),
                       {"data": []}, {"data": [valid["data"][0], valid["data"][0]]},
                       {"data": [{**valid["data"][0], "cwd": "/other"}]},
                       {"data": [{"cwd": str(self.repo), "skills": [self.skill]}]}]:
            with self.subTest(result=result), self.assertRaises(ValueError):
                native.assert_discovery(result, self.repo, self.instruction, "installed")

    def test_negative_controls_reject_stale_success_and_require_specific_malformed_diagnostic(self):
        for state in ["baseline", "missing", "unregistered", "malformed"]:
            with self.subTest(state=state), self.assertRaises(ValueError):
                native.assert_discovery(self.result([self.skill]), self.repo, self.instruction, state)
        for error in [None, {"path": "/other/SKILL.md", "message": "invalid"},
                      {"path": str(self.instruction), "message": ""}]:
            with self.subTest(error=error), self.assertRaises(ValueError):
                native.assert_discovery(self.result(errors=[error] if error else []), self.repo, self.instruction, "malformed")
        native.assert_discovery(self.result(errors=[{"path": str(self.instruction), "message": "invalid YAML"}]),
                                self.repo, self.instruction, "malformed")
        for state in ["baseline", "missing", "unregistered"]:
            native.assert_discovery(self.result(), self.repo, self.instruction, state)
        with self.assertRaisesRegex(ValueError, "unknown discovery state"):
            native.assert_discovery(self.result(), self.repo, self.instruction, "misspelled-state")
        # Codex may include its own bundled system skills; they cannot satisfy
        # the expected user/repository package, even if their names collide.
        system = self.result([{**self.skill, "scope": "system"}])
        native.assert_discovery(system, self.repo, self.instruction, "missing")
        with self.assertRaises(ValueError):
            native.assert_discovery(system, self.repo, self.instruction, "installed")

    def test_child_environment_excludes_credentials_and_resolves_five_independent_targets(self):
        with tempfile.TemporaryDirectory() as temporary, patch.dict(os.environ, {
                "OPENAI_API_KEY": "never-inherit-fixture", "GH_TOKEN": "never-inherit-fixture",
                "CODEX_HOME": "/maintainer/home", "CODEX_MANAGED_CONFIG_TOML": "never-inherit-fixture"}):
            for case, relative, explicit in [
                ("default-user", "home/.codex/skills", False),
                ("explicit-user", "home/.agents/skills", True),
                ("explicit-repository", "repository/.agents/skills", True),
                ("codex-home", "legacy codex home/skills", False),
                ("unregistered-target", "unregistered skills", True),
            ]:
                root = Path(temporary) / case
                env, repo, target, flag = native.case_environment(root, case)
                self.assertEqual(target, root / relative)
                self.assertEqual(flag, explicit)
                self.assertTrue((repo / ".git").is_dir())
                self.assertEqual(Path(env["HOME"]), root / "home")
                self.assertFalse({"OPENAI_API_KEY", "GH_TOKEN", "CODEX_MANAGED_CONFIG_TOML"} & env.keys())
                self.assertEqual("CODEX_HOME" in env, case == "codex-home")
                if case == "codex-home":
                    self.assertTrue(Path(env["CODEX_HOME"]).is_dir())

    def test_archive_is_digest_bound_before_binary_execution(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            archive = root / "client.tar.gz"
            archive.write_bytes(b"substituted archive")
            with self.assertRaisesRegex(ValueError, "checksum mismatch"):
                native.pinned_client(root, archive, {})
            self.assertFalse(any(path.name.startswith("codex-") and path.suffix != ".gz" for path in root.iterdir()))

    def test_cached_archive_replacement_cannot_change_the_verified_snapshot(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            cached = root / "cached.tar.gz"

            def write_archive(payload):
                with tarfile.open(cached, "w:gz") as output:
                    member = tarfile.TarInfo("codex-test")
                    member.size = len(payload)
                    output.addfile(member, io.BytesIO(payload))

            write_archive(b"reviewed binary")
            actual_digest = native.digest
            expected = actual_digest(cached)

            def substitute_after_hash(path):
                value = actual_digest(path)
                write_archive(b"substituted binary")
                return value

            host = (native.platform.system(), native.platform.machine().lower())
            with patch.dict(native.ARCHIVES, {host: ("test", expected)}), patch.object(native, "digest", side_effect=substitute_after_hash):
                binary = native.pinned_client(root, cached, {})
            self.assertEqual(binary.read_bytes(), b"reviewed binary")
            self.assertNotEqual(actual_digest(cached), expected)

    def test_archive_rejects_links_traversal_extra_members_and_oversize_binary(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            archive = root / "client.tar.gz"
            for name, kind, size, extra in [("../codex-test", tarfile.REGTYPE, 1, False),
                                            ("codex-test", tarfile.SYMTYPE, 0, False),
                                            ("codex-test", tarfile.REGTYPE, 1, True),
                                            ("codex-test", tarfile.REGTYPE, 2, False)]:
                with tarfile.open(archive, "w:gz") as output:
                    member = tarfile.TarInfo(name); member.type = kind; member.size = size
                    output.addfile(member, io.BytesIO(b"xx"))
                    if extra:
                        output.addfile(tarfile.TarInfo("extra"))
                host = (native.platform.system(), native.platform.machine().lower())
                with patch.dict(native.ARCHIVES, {host: ("test", native.digest(archive))}), patch.object(native, "MAX_BINARY", 1):
                    with self.subTest(name=name, kind=kind, size=size, extra=extra), self.assertRaises(ValueError):
                        native.pinned_client(root, archive, {})
            self.assertFalse((root / "codex-test").exists())

    def test_failure_report_survives_failed_client_prerequisite(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            report = root / "failure.json"
            with contextlib.redirect_stdout(io.StringIO()), patch.object(check.platform, "system", return_value="Linux"), patch.object(check.platform, "machine", return_value="x86_64"):
                self.assertEqual(check.main(["--cli", str(root / "missing"), "--report", str(report)]), 1)
            result = json.loads(report.read_text())
            self.assertEqual((result["status"], result["installer_kind"], result["failure_stage"]), ("failed", "candidate", "installer"))
            self.assertEqual(result["cases"], [])

    def test_orchestration_exercises_default_and_explicit_targets_and_all_control_states(self):
        # This models the checker lifecycle, not native-client adoption. The live
        # run supplies the separate client-reported evidence for these states.
        with tempfile.TemporaryDirectory() as temporary:
            for case in native.CASES:
                root = Path(temporary) / case
                environment, repo, target, explicit = native.case_environment(root, case)
                package = target / native.PACKAGE_ID
                instruction = package / "SKILL.md"
                source = b"---\nname: implementation-strategy\ndescription: Plan changes.\n---\nInstructions.\n"
                commands = []
                states = []

                def install(command, **kwargs):
                    commands.append(command)
                    self.assertEqual(kwargs["env"], environment)
                    self.assertEqual(kwargs["cwd"], repo)
                    package.mkdir(parents=True)
                    instruction.write_bytes(source)
                    return type("Completed", (), {"returncode": 0, "stderr": b""})()

                def list_skills(_cwd):
                    self.assertEqual(_cwd, repo)
                    skills, errors = [], []
                    if not package.exists():
                        states.append("baseline")
                    elif case == "unregistered-target":
                        states.append("unregistered")
                    elif not instruction.exists():
                        states.append("missing")
                    elif instruction.read_bytes() != source:
                        states.append("malformed")
                        errors = [{"path": str(instruction), "message": "invalid skill"}]
                    else:
                        states.append("installed" if len(states) == 1 else "restored")
                        skills = [{**self.skill, "path": str(instruction),
                                   "scope": "repo" if case == "explicit-repository" else "user"}]
                    return {"data": [{"cwd": str(repo), "skills": skills, "errors": errors}]}

                from unittest.mock import MagicMock
                client = MagicMock()
                client.__enter__.return_value = client
                client.skills.side_effect = list_skills
                with patch.object(native, "case_environment", return_value=(environment, repo, target, explicit)), \
                        patch.object(native, "AppServer", return_value=client), \
                        patch.object(check.subprocess, "run", side_effect=install), \
                        patch.object(check.published, "verify_installed", return_value="payload") as verify:
                    result = {"case": case}
                    check.verify_case(Path("/installer"), Path("/codex"), root, check.published.REGISTRY,
                                      {"SKILL.md": source}, {"sha256": "a" * 64, "manifest_sha256": "b" * 64},
                                      Path("/verifier"), Path("/pinned/skill.yaml"), result)
                self.assertEqual(len(commands), 1)
                self.assertEqual("--target" in commands[0], explicit)
                if explicit:
                    self.assertEqual(commands[0][commands[0].index("--target") + 1], str(target))
                self.assertIn("engineering/implementation-strategy@0.1.0", commands[0])
                self.assertEqual(states, ["baseline", "unregistered"] if case == "unregistered-target"
                                 else ["baseline", "installed", "missing", "malformed", "restored"])
                self.assertEqual(instruction.read_bytes(), source)
                self.assertEqual(verify.call_count, 1 if case == "unregistered-target" else 2)
                self.assertEqual(result["status"], "passed")

    def test_native_discovery_cannot_mutate_installed_bytes_without_failing(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            package = root / "target" / native.PACKAGE_ID
            package.mkdir(parents=True)
            instruction = package / "SKILL.md"
            instruction.write_text("untouched")
            from unittest.mock import MagicMock
            client = MagicMock()
            client.__enter__.return_value = client

            def mutate(_):
                instruction.write_text("unexpected client effect")
                return self.result()

            client.skills.side_effect = mutate
            with patch.object(native, "case_environment", return_value=({}, self.repo, root / "target", False)), \
                    patch.object(native, "AppServer", return_value=client), \
                    self.assertRaisesRegex(ValueError, "mutated the installed package"):
                check.verify_case(Path("/installer"), Path("/codex"), root, check.published.REGISTRY, {}, {},
                                  Path("/verifier"), Path("/manifest"), {"case": "default-user"})


class ProtocolBoundaryTests(unittest.TestCase):
    def server(self, root, body):
        binary = root / "fake-client"
        binary.write_text(f"#!{sys.executable}\nimport json, sys, time\n" + body)
        binary.chmod(0o755)
        return binary

    def test_notification_is_not_success_and_partial_lines_are_supported(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            binary = self.server(root, "request=json.loads(sys.stdin.readline())\n"
                                 "print(json.dumps({'method':'status','params':{}}),flush=True)\n"
                                 "payload=json.dumps({'id':request['id'],'result':{'data':[]}})+'\\n'\n"
                                 "sys.stdout.write(payload[:5]);sys.stdout.flush();time.sleep(.02)\n"
                                 "sys.stdout.write(payload[5:]);sys.stdout.flush()\n"
                                 "sys.stdin.read()\n")
            with native.AppServer(binary, root, {"PATH": os.defpath}, timeout=2) as client:
                self.assertEqual(client.request("skills/list", {}), {"data": []})
                with self.assertRaisesRegex(ValueError, "cannot start a turn"):
                    client.request("turn/start", {})
                with self.assertRaisesRegex(ValueError, "cannot start a turn"):
                    client.send({"id": 9, "method": "turn/start", "params": {}})
            self.assertIsNotNone(client.process.poll())

    def test_rpc_failures_are_bounded_and_never_converted_to_discovery_success(self):
        cases = {
            "wrong-id": "print('{\"id\":99,\"result\":{}}',flush=True)",
            "invalid-json": "print('not-json',flush=True)",
            "non-object": "print('[]',flush=True)",
            "error": "print('{\"id\":1,\"error\":{\"message\":\"rejected\"}}',flush=True)",
            "missing-result": "print('{\"id\":1}',flush=True)",
            "exit": "sys.exit(0)",
            "hang": "time.sleep(10)",
            "oversize": "print('x'*2048,flush=True)",
            "stderr-flood": "sys.stderr.write('x'*8192);sys.stderr.flush();time.sleep(10)",
        }
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for case, behavior in cases.items():
                binary = self.server(root, "sys.stdin.readline()\n" + behavior + "\n")
                with self.subTest(case=case), patch.object(native, "MAX_MESSAGE", 1024), patch.object(native, "MAX_OUTPUT", 4096):
                    with self.assertRaises((ValueError, OSError)):
                        with native.AppServer(binary, root, {"PATH": os.defpath}, timeout=.3) as client:
                            client.request("skills/list", {})
                    self.assertIsNotNone(client.process.poll())

    def test_initialize_rejects_wrong_version_and_home(self):
        with tempfile.TemporaryDirectory() as temporary:
            root = Path(temporary)
            for result in [{"userAgent": "skills_hub_discovery_check/0.1.0 test", "codexHome": str(root)},
                           {"userAgent": f"skills_hub_discovery_check/{native.VERSION} test", "codexHome": "/real/home"}]:
                binary = self.server(root, "sys.stdin.readline()\nprint(" + repr(json.dumps({"id": 1, "result": result})) + ",flush=True)\nsys.stdin.read()\n")
                with self.subTest(result=result), self.assertRaises(ValueError):
                    with native.AppServer(binary, root, {"PATH": os.defpath}) as client:
                        client.initialize(root)


if __name__ == "__main__":
    unittest.main()
