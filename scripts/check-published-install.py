#!/usr/bin/env python3
"""Bounded #52 proof: pinned published instruction or GA4 package, no provider calls."""

import argparse
import datetime
import hashlib
import io
import importlib.util
import json
import os
from pathlib import Path, PurePosixPath
import platform
import re
import shutil
import socket
import stat
import subprocess
import tarfile
import tempfile
import urllib.parse
import urllib.request


REPOSITORY = "ai-knowledge-hub/ai-skills-guide"
PACKAGE = "engineering/implementation-strategy"
VERSION = "0.1.0"
REGISTRY = "https://skills.ai-knowledge-hub.org/registry/skills-index.json"
RECEIPT = ".skills-hub-install.json"
MAX_DOWNLOAD = 32 * 1024 * 1024
DEFAULT_VERIFIER = Path(__file__).resolve().parents[1] / "bin" / "published-install-verifier"
INSTRUCTIONS = {"module": "skills", "id": PACKAGE, "version": VERSION, "manifest": "skill.yaml",
                "registry": REGISTRY, "repair": "SKILL.md", "required": {"SKILL.md", "README.md", "skill.yaml"}}
GA4 = {"module": "tools", "id": "analytics/ga4-mcp-connector", "version": "0.2.0", "manifest": "tool.yaml",
       "registry": "https://skills.ai-knowledge-hub.org/registry/tools-index.json",
       "repair": "scripts/ga4_mcp_server.mjs",
       "required": {"tool.yaml", "README.md", "TOOL.md", ".mcp.json", "scripts/ga4_mcp_server.mjs"}}
SAMPLES = {"instructions": INSTRUCTIONS, "ga4": GA4}
spec = importlib.util.spec_from_file_location("ga4_startup", Path(__file__).with_name("ga4_mcp_startup.py"))
ga4_startup = importlib.util.module_from_spec(spec)
spec.loader.exec_module(ga4_startup)


def require(condition, message):
    if not condition:
        raise ValueError(message)


def sha256(data):
    return hashlib.sha256(data).hexdigest()


def https_origin(url):
    parsed = urllib.parse.urlsplit(url)
    require(parsed.scheme == "https" and parsed.hostname and not parsed.username
            and not parsed.password and not parsed.fragment, "expected a public HTTPS URL without credentials or fragment")
    return parsed.scheme, parsed.hostname, parsed.port or 443


class HTTPSRedirects(urllib.request.HTTPRedirectHandler):
    def __init__(self, origin=None):
        self.origin = origin

    def redirect_request(self, req, fp, code, msg, headers, newurl):
        origin = https_origin(newurl)
        require(self.origin is None or origin == self.origin, "cross-origin registry/artifact redirect rejected")
        return super().redirect_request(req, fp, code, msg, headers, newurl)


def download(url, origin=None, headers=None):
    actual = https_origin(url)
    require(origin is None or actual == origin, "registry and package must have the same HTTPS origin")
    request = urllib.request.Request(url, headers={"User-Agent": "skills-hub-published-install-check", "Accept": "*/*", **(headers or {})})
    opener = urllib.request.build_opener(HTTPSRedirects(origin))
    with opener.open(request, timeout=30) as response:
        data = response.read(MAX_DOWNLOAD + 1)
    require(len(data) <= MAX_DOWNLOAD, "download exceeds the bounded verification limit")
    return data


def release_asset_names(tag, system, machine):
    require(re.fullmatch(r"v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?", tag), "an exact vX.Y.Z release tag is required")
    os_name = {"Darwin": "darwin", "Linux": "linux"}.get(system)
    arch = {"arm64": "arm64", "aarch64": "arm64", "x86_64": "amd64", "AMD64": "amd64"}.get(machine)
    require(os_name and arch, "this check supports macOS/Linux on amd64/arm64")
    return f"skills-hub_{tag}_{os_name}_{arch}", "SHA256SUMS", f"skills-hub_{tag}_provenance.json"


def released_cli(tag, directory, report):
    names = release_asset_names(tag, platform.system(), platform.machine())
    metadata_url = f"https://api.github.com/repos/{REPOSITORY}/releases/tags/{tag}"
    token = os.environ.get("GH_TOKEN")
    # CI's read-only token avoids shared-runner anonymous API rate limits. Bind
    # it to api.github.com; never attach it to asset downloads or cross-origin redirects.
    metadata = json.loads(download(metadata_url, origin=https_origin(metadata_url),
                                   headers={"Authorization": f"Bearer {token}"} if token else None))
    require(metadata.get("tag_name") == tag and not metadata.get("draft"), "release tag does not identify a published release")
    assets = {asset["name"]: asset["browser_download_url"] for asset in metadata.get("assets", [])}
    report["release"] = {"tag": tag, "url": metadata.get("html_url"), "assets": sorted(assets)}
    missing = [name for name in names if name not in assets]
    require(not missing, "published CLI assets missing: " + ", ".join(missing))
    for name in names:
        expected_url = f"https://github.com/{REPOSITORY}/releases/download/{tag}/{name}"
        require(assets[name] == expected_url, "release asset URL does not match the selected repository/tag")
        (directory / name).write_bytes(download(expected_url))
    binary, sums, bundle = (directory / name for name in names)
    matches = [line.split()[0] for line in sums.read_text().splitlines()
               if len(line.split()) == 2 and line.split()[1].lstrip("*") == binary.name]
    require(matches == [sha256(binary.read_bytes())], "released CLI checksum mismatch or ambiguous checksum record")
    # Check provenance before granting the downloaded binary execute permission.
    subprocess.run([
        "gh", "attestation", "verify", str(binary), "--bundle", str(bundle),
        "--repo", REPOSITORY, "--signer-workflow", f"{REPOSITORY}/.github/workflows/release-on-tag.yml",
        "--source-ref", f"refs/tags/{tag}", "--deny-self-hosted-runners",
    ], check=True, timeout=120, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    report["release"]["checksum_and_provenance"] = "passed"
    binary.chmod(0o755)
    return binary


def selected_release(index, sample=INSTRUCTIONS):
    require(index.get("registry_version") == "1.3", "published registry must use version-bound 1.3 records")
    entries = [entry for entry in index.get("skills", []) if entry.get("id") == sample["id"]]
    require(len(entries) == 1, "pinned package must occur exactly once")
    entry = entries[0]
    versions = [item for item in entry.get("versions", []) if item.get("version") == sample["version"]]
    require(len(versions) == 1, "pinned package release must occur exactly once")
    version = versions[0]
    for key in ["sha256", "manifest_sha256"]:
        require(re.fullmatch(r"[a-f0-9]{64}", version.get(key, "")), f"invalid {key} in published release")
    return version


def archive_files(data, sample=INSTRUCTIONS, modes=None):
    files = {}
    seen = set()
    total = 0
    with tarfile.open(fileobj=io.BytesIO(data), mode="r:gz") as archive:
        for member in archive:
            name = member.name.removeprefix("./").rstrip("/")
            path = PurePosixPath(name)
            require(name and not path.is_absolute() and ".." not in path.parts and "\\" not in name,
                    "unsafe archive member")
            require(name not in seen and len(seen) < 1024, "duplicate or excessive archive members")
            seen.add(name)
            require(member.isfile() or member.isdir(), "archive links and special files are unsupported")
            total += member.size
            require(total <= MAX_DOWNLOAD, "expanded archive exceeds verification limit")
            if member.isfile():
                require(sample == GA4 or not member.mode & 0o111, "instruction sample unexpectedly contains executable files")
                files[name] = archive.extractfile(member).read()
            if modes is not None:
                modes[name] = 0o755 if member.isdir() or member.mode & 0o111 else 0o644
                for parent in path.parents:
                    if parent != PurePosixPath("."):
                        modes[parent.as_posix()] = 0o755
    require(sample["required"] <= files.keys(), "pinned sample is incomplete")
    require(RECEIPT not in files, "release must not supply its own install receipt")
    return files


def inspect_pinned(verifier, manifest_path, manifest_sha, package_dir=None, runtime=None, artifact_sha=None, sample=INSTRUCTIONS):
    args = [str(verifier), "--manifest", str(manifest_path), "--manifest-sha256", manifest_sha,
            "--id", sample["id"], "--version", sample["version"], "--module", sample["module"]]
    if package_dir is not None:
        args += ["--package-dir", str(package_dir), "--runtime", runtime, "--artifact-sha256", artifact_sha]
    result = subprocess.run(args, env={"PATH": os.defpath}, capture_output=True, timeout=30)
    require(result.returncode == 0, "pinned manifest/receipt verification failed: " + result.stderr.decode(errors="replace")[-2000:])
    document = json.loads(result.stdout)
    require(document.get("id") == sample["id"] and document.get("version") == sample["version"], "verifier returned another pinned subject")
    if package_dir is not None:
        require(document.get("receipt_verified") is True, "runtime receipt verification did not pass")
    return document


def filesystem_snapshot(root):
    """Use lstat: never follow a changed link while comparing duplicate effects."""
    require(root.exists() and not root.is_symlink(), "installed root was removed or replaced by a link")
    snapshot = {}

    def visit(path):
        metadata = path.lstat()
        mode = metadata.st_mode
        name = "." if path == root else path.relative_to(root).as_posix()
        # Access time is omitted because reading the payload can update it.
        value = [stat.S_IFMT(mode), stat.S_IMODE(mode), metadata.st_uid, metadata.st_gid,
                 metadata.st_ino, metadata.st_nlink, metadata.st_mtime_ns, metadata.st_ctime_ns]
        if stat.S_ISREG(mode):
            value.append(path.read_bytes())
        elif stat.S_ISLNK(mode):
            value.append(os.readlink(path))
        snapshot[name] = value
        if stat.S_ISDIR(mode):
            for child in path.iterdir():
                visit(child)

    visit(root)
    return snapshot


def verify_installed(package_dir, expected_files, runtime, artifact_sha, verifier, manifest_path, manifest_sha,
                     sample=INSTRUCTIONS, modes=None):
    require(package_dir.is_dir() and not package_dir.is_symlink(), "CLI reported success without an installed package")
    actual = {}
    actual_modes = {}
    for path in package_dir.rglob("*"):
        require(not path.is_symlink(), "installed package contains a symlink")
        require(path.is_dir() or path.is_file(), "installed package contains a special file")
        relative = path.relative_to(package_dir).as_posix()
        if relative != RECEIPT:
            actual_modes[relative] = stat.S_IMODE(path.stat().st_mode)
        if path.is_file():
            require(sample == GA4 or not path.stat().st_mode & 0o111, "installed instruction file unexpectedly became executable")
            actual[relative] = path.read_bytes()
    receipt = json.loads(actual.pop(RECEIPT, b"{}"))
    for key, value in {"schema_version": "skills-hub.install-receipt/v1", "source": "remote", "module": sample["module"],
                       "id": sample["id"], "version": sample["version"], "runtime": runtime, "artifact_sha256": artifact_sha}.items():
        require(receipt.get(key) == value, f"installed receipt has incorrect {key}")
    require(actual == expected_files, "installed file inventory/bytes differ from the downloaded release")
    if sample == GA4:
        require(modes is not None and actual_modes == modes, "installed GA4 inventory/permissions differ from the downloaded release")
    inspect_pinned(verifier, manifest_path, manifest_sha, package_dir, runtime, artifact_sha, sample)
    return sha256(json.dumps({name: sha256(data) for name, data in sorted(actual.items())}).encode())


def verify_runtime(binary, registry_url, runtime, directory, files, artifact_sha, result, verifier, manifest_path, manifest_sha,
                   sample=INSTRUCTIONS, modes=None):
    work = directory / runtime
    work.mkdir()
    target = work / "runtime" / sample["module"]
    cache = work / "cache"
    package_dir = target / sample["id"]
    args = [str(binary), "install", "--source", "remote", "--module", sample["module"], "--registry-url", registry_url,
            "--entry", f"{sample['id']}@{sample['version']}", "--runtime", runtime, "--target", str(target), "--cache-dir", str(cache)]
    if sample == GA4:
        args += ["--execution-runtime", "node22"]
    # No adopting-client credentials, runtime homes or checkout/package paths are
    # supplied. Explicit target/cache paths keep all writes in disposable space.
    environment = {"PATH": os.defpath}
    completed = subprocess.run(args, cwd=work, env=environment, capture_output=True, timeout=150)
    require(completed.returncode == 0, "cold remote install failed: " + completed.stderr.decode(errors="replace")[-2000:])
    result["payload_sha256"] = verify_installed(package_dir, files, runtime, artifact_sha, verifier, manifest_path, manifest_sha, sample, modes)
    result["receipt_verified"] = True
    result["cold_install"] = "passed"
    if sample == GA4:
        before_startup = filesystem_snapshot(package_dir)
        result["startup"] = ga4_startup.verify(binary, completed.stdout, package_dir, work)
        require(filesystem_snapshot(package_dir) == before_startup, "GA4 startup mutated installed files or receipt")
    before = filesystem_snapshot(package_dir)
    duplicate = subprocess.run(args, cwd=work, env=environment, capture_output=True, timeout=150)
    after = filesystem_snapshot(package_dir)
    require(duplicate.returncode != 0 and before == after, "duplicate install did not reject safely without --force")
    result["duplicate_rejected_without_mutation"] = "passed"
    (package_dir / sample["repair"]).write_text("deliberately damaged disposable installation\n")
    # An unavailable local proxy makes accidental ordinary HTTP(S) traffic fail
    # during the cache-only reinstall. Existing installer tests cover its HTTP
    # transport independently; this is not an OS-wide network sandbox.
    with socket.socket() as blocked:
        blocked.bind(("127.0.0.1", 0))
        proxy = f"http://127.0.0.1:{blocked.getsockname()[1]}"
        offline_env = {**environment, "HTTP_PROXY": proxy, "HTTPS_PROXY": proxy, "NO_PROXY": ""}
        offline = subprocess.run(args + ["--offline", "--force"], cwd=work, env=offline_env, capture_output=True, timeout=150)
    require(offline.returncode == 0, "offline forced reinstall failed: " + offline.stderr.decode(errors="replace")[-2000:])
    verify_installed(package_dir, files, runtime, artifact_sha, verifier, manifest_path, manifest_sha, sample, modes)
    result["offline_reinstall_restored_release_bytes"] = "passed"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    client = parser.add_mutually_exclusive_group(required=True)
    client.add_argument("--cli", type=Path, help="explicit candidate binary; never recorded as a released client")
    client.add_argument("--cli-release", help="exact GitHub release tag; requires binaries, checksums, provenance and gh")
    parser.add_argument("--sample", choices=SAMPLES, default="instructions")
    parser.add_argument("--registry-url", help="public HTTPS index; defaults to the selected sample's module index")
    parser.add_argument("--verifier", type=Path, default=DEFAULT_VERIFIER, help="read-only runtime manifest/receipt verifier")
    parser.add_argument("--report", type=Path, required=True)
    parser.add_argument("--runtimes", nargs="+", choices=["codex", "claude", "generic"], default=["codex", "claude", "generic"])
    args = parser.parse_args()
    sample = SAMPLES[args.sample]
    args.registry_url = args.registry_url or sample["registry"]
    is_release = args.cli_release is not None
    report = {"schema_version": "published-install-check/v1", "checked_at": datetime.datetime.now(datetime.timezone.utc).isoformat(),
              "status": "failed", "client_kind": "release" if is_release else "candidate",
              "host": {"os": platform.system(), "architecture": platform.machine()},
              "registry_url": args.registry_url, "sample": args.sample,
              "package": f"{sample['id']}@{sample['version']}", "module": sample["module"], "runtimes": []}
    stage = "client"
    try:
        with tempfile.TemporaryDirectory(prefix="skills52-clean-") as temporary:
            clean = Path(temporary)
            bin_dir = clean / "bin"
            bin_dir.mkdir()
            if is_release:
                binary = released_cli(args.cli_release, bin_dir, report)
            else:
                require(args.cli.is_file(), "candidate binary is missing")
                binary = bin_dir / "skills-hub-candidate"
                shutil.copyfile(args.cli.resolve(), binary)
                binary.chmod(0o755)
            report["cli_sha256"] = sha256(binary.read_bytes())
            stage = "published-release"
            origin = https_origin(args.registry_url)
            registry_bytes = download(args.registry_url, origin)
            version = selected_release(json.loads(registry_bytes), sample)
            report["registry_sha256"] = sha256(registry_bytes)
            report["selected_release"] = version
            archive = download(version["artifact_url"], origin)
            manifest = download(version["manifest_url"], origin)
            require(sha256(archive) == version["sha256"], "published archive checksum mismatch")
            require(sha256(manifest) == version["manifest_sha256"], "published manifest checksum mismatch")
            modes = {}
            files = archive_files(archive, sample, modes)
            require(files[sample["manifest"]] == manifest, "served manifest differs from the archive manifest")
            pinned_dir = clean / "pinned-release"
            pinned_dir.mkdir()
            for name, data in files.items():
                path = pinned_dir / name
                path.parent.mkdir(parents=True, exist_ok=True)
                path.write_bytes(data)
                path.chmod(modes[name])
            require(args.verifier.is_file(), "build bin/published-install-verifier before running this check")
            verifier = bin_dir / "published-install-verifier"
            shutil.copyfile(args.verifier.resolve(), verifier)
            verifier.chmod(0o755)
            report["verifier_sha256"] = sha256(verifier.read_bytes())
            manifest_path = pinned_dir / sample["manifest"]
            pinned = inspect_pinned(verifier, manifest_path, version["manifest_sha256"], sample=sample)
            report["pinned_manifest"] = pinned
            report["published_files"] = {name: sha256(data) for name, data in sorted(files.items())}
            for runtime in args.runtimes:
                stage = f"install-{runtime}"
                require(runtime in pinned.get("runtimes", []), f"pinned sample does not declare {runtime}")
                result = {"runtime": runtime}
                report["runtimes"].append(result)
                verify_runtime(binary, args.registry_url, runtime, clean, files, version["sha256"], result,
                               verifier, manifest_path, version["manifest_sha256"], sample, modes)
        report["status"] = "passed"
    except (OSError, ValueError, KeyError, TypeError, AttributeError, tarfile.TarError, subprocess.SubprocessError) as error:
        report["failure_stage"] = stage
        report["error"] = str(error)
    args.report.parent.mkdir(parents=True, exist_ok=True)
    args.report.write_text(json.dumps(report, indent=2) + "\n")
    print(f"{report['status']}: {report['client_kind']} client; report: {args.report}")
    if report["status"] != "passed":
        print(report["error"])
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
