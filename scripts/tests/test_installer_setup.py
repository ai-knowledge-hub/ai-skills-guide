"""Follow the retained GA4 README's replacement path without provider access."""

import hashlib
import io
import json
import os
from pathlib import Path, PurePosixPath
import re
import shutil
import subprocess
import tarfile
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
RELEASE = ROOT / "releases/tools-mcp/analytics/ga4-mcp-connector/0.2.0"


class InstallerSetupTests(unittest.TestCase):
    def test_retained_readme_redirects_to_offline_guide_and_launchable_registration(self):
        node = shutil.which("node")
        self.assertIsNotNone(node, "this integration test requires Node.js 22+")
        version = subprocess.check_output([node, "--version"], text=True).strip()
        self.assertGreaterEqual(int(version.lstrip("v").split(".")[0]), 22)
        archive_before = (RELEASE / "package.tar.gz").read_bytes()
        with tempfile.TemporaryDirectory(prefix="installer-setup-") as temporary:
            root = Path(temporary)
            binary = root / "skills-hub"
            subprocess.run(["go", "build", "-tags", "installer_release", "-o", str(binary),
                            "./cmd/skills-hub"], cwd=ROOT, check=True, capture_output=True)
            package_root = root / "source/analytics/ga4-mcp-connector"
            expected = {}
            with tarfile.open(fileobj=io.BytesIO(archive_before), mode="r:gz") as archive:
                for member in archive:
                    name = PurePosixPath(member.name)
                    self.assertFalse(name.is_absolute() or ".." in name.parts)
                    self.assertTrue(member.isdir() or member.isfile())
                    destination = package_root / name
                    if member.isdir():
                        destination.mkdir(parents=True, exist_ok=True)
                    else:
                        data = archive.extractfile(member).read()
                        expected[name.as_posix()] = data
                        destination.parent.mkdir(parents=True, exist_ok=True)
                        destination.write_bytes(data)
                        destination.chmod(member.mode & 0o777)
            self.assertIn(b"./bin/skills-hub auth configure", expected["README.md"],
                          "the regression must retain the actual historical dead end")
            entry = json.loads((RELEASE / "registry-entry.json").read_text())
            entry.update(latest="0.2.0", versions=[{"version": "0.2.0"}])
            registry = root / "registry.json"
            registry.write_text(json.dumps({"registry_version": "1.3", "skills": [entry]}))
            target = root / "target"
            working = root / "outside-checkout"
            working.mkdir()
            environment = {"PATH": os.environ["PATH"], "HOME": str(root)}
            installed = subprocess.run([
                str(binary), "install", "--source", "local", "--module", "tools",
                "--entry", "analytics/ga4-mcp-connector@0.2.0", "--registry", str(registry),
                "--root", str(root / "source"), "--runtime", "generic", "--target", str(target)
            ], cwd=working, env=environment, capture_output=True, text=True, check=True)
            # Follow the actual emitted command, using the very same installer binary.
            command = re.search(r"^install.setup_guidance: (.+)$", installed.stdout, re.M)
            self.assertIsNotNone(command, installed.stdout)
            self.assertEqual(command.group(1).split(), ["skills-hub", "help", "setup"])
            # A user following the delivered README's old command gets the same
            # usable replacement, rather than another unavailable-command dead end.
            legacy = subprocess.run([str(binary), "auth", "configure",
                                     "analytics/ga4-mcp-connector@0.2.0", "--module", "tools"],
                                    cwd=working, env=environment, capture_output=True, text=True)
            self.assertNotEqual(legacy.returncode, 0)
            self.assertIn(command.group(1), legacy.stderr)
            guide = subprocess.check_output([str(binary), *command.group(1).split()[1:]],
                                            cwd=working, env=environment, text=True)
            self.assertIn("supersedes retained README", guide)
            self.assertIn("GA4_CREDENTIAL_SOURCE", guide)
            self.assertIn("forward its value under", guide)
            self.assertIn("GA4_CREDENTIAL", guide)
            self.assertIn("GA4_ALLOWED_PROPERTY_IDS", guide)
            blocks = re.findall(r"```json\n(.*?)\n```", guide, re.S)
            self.assertEqual(len(blocks), 1, "one unambiguous MCP registration example")
            real_package = target / "analytics/ga4-mcp-connector"
            package = root / "installed-ga4-link"
            package.symlink_to(real_package, target_is_directory=True)
            registration = json.loads(blocks[0].replace("<absolute-installed-ga4-directory>", str(package)))
            server = registration["mcpServers"]["ga4"]
            self.assertEqual(server["type"], "stdio")
            self.assertEqual(server["command"], "node")
            self.assertEqual(server["args"], ["--preserve-symlinks-main", str(package / "scripts/ga4_mcp_server.mjs")])
            for name, data in expected.items():
                self.assertEqual((package / name).read_bytes(), data, name)
            snapshot = {p.relative_to(package).as_posix(): hashlib.sha256(p.read_bytes()).hexdigest()
                        for p in package.rglob("*") if p.is_file()}
            self.assertEqual(set(snapshot), set(expected) | {".skills-hub-install.json"})
            # The retained server uses fetch for provider requests. Reject any such
            # request while executing the guide's credential-free launch check.
            blocker = root / "block-provider.mjs"
            blocker.write_text("globalThis.fetch = () => { throw new Error('provider request forbidden in setup test'); };\n")
            launch_environment = {**environment, "NODE_OPTIONS": "--import=" + str(blocker)}
            launched = subprocess.run([node, *server["args"], "--healthcheck"], cwd=working,
                                      env=launch_environment, capture_output=True, text=True,
                                      check=True, timeout=15)
            self.assertEqual(json.loads(launched.stdout), {
                "status": "ok", "server": "ga4-mcp-connector", "version": "0.2.0", "transport": "stdio"
            })
            self.assertEqual(snapshot, {
                p.relative_to(package).as_posix(): hashlib.sha256(p.read_bytes()).hexdigest()
                for p in package.rglob("*") if p.is_file()
            }, "setup guidance/launch must not modify payloads or their install receipt")
        self.assertEqual((RELEASE / "package.tar.gz").read_bytes(), archive_before)
