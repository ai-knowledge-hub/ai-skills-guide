"""Pinned, credential-free Codex discovery oracle for the bounded #52 check."""

import hashlib
import json
import os
from pathlib import Path
import platform
import selectors
import subprocess
import tarfile
import time
import urllib.request


VERSION = "0.168.0"
RELEASE = "https://github.com/openai/codex/releases/download/rust-v" + VERSION
# Reviewed GitHub release asset digests; neither latest nor runner-installed Codex
# selects the client under test. A version change requires reviewing these pins.
ARCHIVES = {
    ("Darwin", "arm64"): ("aarch64-apple-darwin", "c94dffc075ce5527a55d060f35412b45d1c1c1d16306a337dcf0d8e3ecfebc98"),
    ("Darwin", "x86_64"): ("x86_64-apple-darwin", "01511787bdecab86e03f55fd390a5998590c7ca5c80fc64f2553b01f14ff9057"),
    ("Linux", "x86_64"): ("x86_64-unknown-linux-musl", "b983beb86dcc606cead6ee226ae2ab08e7e6eee07df4cb8d16f441f107b9e2ef"),
    ("Linux", "aarch64"): ("aarch64-unknown-linux-musl", "d274cbbcb23593e382e49b29a3f04ee7bcf9603deaf41de93ebb2b5243fec4c1"),
}
SKILL_NAME = "implementation-strategy"
PACKAGE_ID = "engineering/implementation-strategy"
MAX_ARCHIVE = 256 * 1024 * 1024
MAX_BINARY = 512 * 1024 * 1024
MAX_MESSAGE = 1024 * 1024
MAX_OUTPUT = 8 * 1024 * 1024
CASES = ("default-user", "explicit-user", "explicit-repository", "codex-home", "unregistered-target")


def require(condition, message):
    if not condition:
        raise ValueError(message)


def digest(path):
    with path.open("rb") as stream:
        return hashlib.file_digest(stream, "sha256").hexdigest()


def pinned_client(directory, archive_path, report):
    host = (platform.system(), platform.machine().lower())
    require(host in ARCHIVES, "native discovery supports pinned macOS/Linux amd64/arm64 clients only")
    target, expected = ARCHIVES[host]
    name = f"codex-{target}"
    url = f"{RELEASE}/{name}.tar.gz"
    report.update(version=VERSION, archive_url=url, archive_sha256=expected)
    staged = directory / "codex-download.tar.gz"

    def copy_bounded(source):
        with staged.open("wb") as output:
            size = 0
            while chunk := source.read(1024 * 1024):
                size += len(chunk)
                require(size <= MAX_ARCHIVE, "Codex archive exceeds download limit")
                output.write(chunk)

    if archive_path is None:
        with urllib.request.urlopen(url, timeout=60) as response:
            copy_bounded(response)
    else:
        require(archive_path.is_file(), "cached Codex archive must be a regular file")
        # Hash and execute a private snapshot, never reopen a caller-owned
        # cached archive after verification (it may change concurrently).
        with archive_path.open("rb") as source:
            copy_bounded(source)
    require(digest(staged) == expected, "pinned Codex archive checksum mismatch")
    binary = directory / name
    with tarfile.open(staged, "r:gz") as archive:
        members = archive.getmembers()
        require(len(members) == 1 and members[0].name == name and members[0].isfile()
                and 0 < members[0].size <= MAX_BINARY, "unexpected pinned Codex archive layout")
        with archive.extractfile(members[0]) as source, binary.open("wb") as output:
            while chunk := source.read(1024 * 1024):
                output.write(chunk)
    binary.chmod(0o755)
    report["binary_sha256"] = digest(binary)
    return binary


def case_environment(root, case):
    """Construct child context from scratch; never load the maintainer's home."""
    home, repo = root / "home", root / "repository"
    home.mkdir(parents=True)
    repo.mkdir()
    temporary = root / "tmp"
    temporary.mkdir()
    environment = {"PATH": os.defpath, "HOME": str(home), "TMPDIR": str(temporary),
                   "XDG_CONFIG_HOME": str(home / ".config"), "XDG_CACHE_HOME": str(home / ".cache"),
                   "XDG_DATA_HOME": str(home / ".local/share"), "RUST_LOG": "error"}
    subprocess.run(["git", "init", "--quiet", str(repo)], check=True, env=environment, capture_output=True, timeout=15)
    target = home / ".codex/skills"
    explicit = case != "default-user" and case != "codex-home"
    if case == "explicit-user":
        target = home / ".agents/skills"
    elif case == "explicit-repository":
        target = repo / ".agents/skills"
    elif case == "codex-home":
        environment["CODEX_HOME"] = str(root / "legacy codex home")
        # The native client requires an explicitly overridden home to exist.
        Path(environment["CODEX_HOME"]).mkdir()
        target = Path(environment["CODEX_HOME"]) / "skills"
    elif case == "unregistered-target":
        target = root / "unregistered skills"
    else:
        require(case == "default-user", "unknown discovery case")
    return environment, repo, target, explicit


class AppServer:
    """Sequential JSON-lines RPC, bounded by time and bytes, with no model turns."""

    def __init__(self, binary, cwd, environment, timeout=30):
        self.timeout = timeout
        self.next_id = 0
        self.pending = b""
        self.stderr = b""
        self.total = 0
        self.selector = selectors.DefaultSelector()
        self.process = subprocess.Popen([str(binary), "app-server", "-c", "analytics.enabled=false",
                                         "-c", "check_for_update_on_startup=false"],
                                        cwd=cwd, env=environment, stdin=subprocess.PIPE,
                                        stdout=subprocess.PIPE, stderr=subprocess.PIPE)
        self.selector.register(self.process.stdout, selectors.EVENT_READ, "stdout")
        self.selector.register(self.process.stderr, selectors.EVENT_READ, "stderr")

    def __enter__(self):
        return self

    def __exit__(self, *_):
        self.process.stdin.close()
        try:
            self.process.wait(timeout=3)
        except subprocess.TimeoutExpired:
            self.process.kill()
            self.process.wait(timeout=3)
        self.process.stdout.close()
        self.process.stderr.close()
        self.selector.close()

    def send(self, message):
        require(message.get("method") in {"initialize", "initialized", "skills/list"},
                "discovery oracle cannot start a turn or execute a skill")
        self.process.stdin.write(json.dumps(message).encode() + b"\n")
        self.process.stdin.flush()

    def request(self, method, params):
        require(method in {"initialize", "skills/list"}, "discovery oracle cannot start a turn or execute a skill")
        self.next_id += 1
        self.send({"id": self.next_id, "method": method, "params": params})
        deadline = time.monotonic() + self.timeout
        while True:
            while b"\n" in self.pending:
                line, self.pending = self.pending.split(b"\n", 1)
                require(len(line) <= MAX_MESSAGE, "Codex response exceeds message limit")
                message = json.loads(line)
                require(isinstance(message, dict), "invalid Codex RPC message")
                if "id" not in message:
                    continue  # Notifications do not establish a request's result.
                require(message["id"] == self.next_id, "unexpected Codex response identity")
                require("error" not in message and isinstance(message.get("result"), dict),
                        f"Codex {method} failed: {message.get('error')}")
                return message["result"]
            remaining = deadline - time.monotonic()
            require(remaining > 0, f"Codex {method} timed out")
            events = self.selector.select(remaining)
            require(events, f"Codex {method} timed out")
            for key, _ in events:
                chunk = os.read(key.fileobj.fileno(), 65536)
                if not chunk:
                    self.selector.unregister(key.fileobj)
                    require(key.data != "stdout", "Codex exited before its response: " + self.stderr.decode(errors="replace")[-2000:])
                    continue
                self.total += len(chunk)
                require(self.total <= MAX_OUTPUT, "Codex output exceeds session limit")
                if key.data == "stdout":
                    self.pending += chunk
                    require(len(self.pending) <= MAX_MESSAGE, "Codex response exceeds message limit")
                else:
                    self.stderr = (self.stderr + chunk)[-2000:]

    def initialize(self, expected_home):
        result = self.request("initialize", {"clientInfo": {"name": "skills_hub_discovery_check", "version": "0.1.0"}})
        require(Path(result.get("codexHome", "")).resolve() == expected_home.resolve(), "Codex used an unexpected home")
        agent = result.get("userAgent", "")
        require(agent.startswith(f"skills_hub_discovery_check/{VERSION} "), "Codex reported an unexpected version")
        self.send({"method": "initialized", "params": {}})
        return result

    def skills(self, cwd):
        return self.request("skills/list", {"cwds": [str(cwd)], "forceReload": True})


def assert_discovery(result, cwd, instruction, state, scope="user"):
    """Name AND exact path are independent oracles, not a success flag or count."""
    require(state in {"baseline", "installed", "missing", "malformed", "restored", "unregistered"},
            "unknown discovery state")
    rows = result.get("data")
    require(isinstance(rows, list) and len(rows) == 1 and rows[0].get("cwd") == str(cwd),
            "Codex returned the wrong working directory")
    skills, errors = rows[0].get("skills"), rows[0].get("errors")
    require(isinstance(skills, list) and isinstance(errors, list), "incomplete Codex discovery result")
    visible = [skill for skill in skills if skill.get("scope") != "system"]
    expected = str(instruction)
    if state in {"installed", "restored"}:
        require(len(visible) == 1 and visible[0].get("name") == SKILL_NAME
                and visible[0].get("path") == expected and visible[0].get("scope") == scope
                and visible[0].get("enabled") is True and bool(visible[0].get("description"))
                and not errors, "Codex did not discover the exact enabled skill name/path/scope")
    else:
        require(not visible, "Codex discovered an absent, malformed or unregistered skill")
        if state == "malformed":
            require(len(errors) == 1 and errors[0].get("path") == expected
                    and bool(errors[0].get("message")), "Codex omitted the malformed skill diagnostic")
        else:
            require(not errors, "unexpected Codex discovery error")
    return {"state": state, "skills": visible, "errors": errors, "status": "passed"}
