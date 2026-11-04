"""Independent bounded oracle for code-maintenance-plugin@0.2.0 delivery."""

import datetime
import hashlib
import json
import os
from pathlib import PurePosixPath
import socket
import stat
import subprocess


ID = "engineering/code-maintenance-plugin"
RECEIPT = ".skills-hub-install.json"
ROOT_CONTRACT = "7c040394d0078f9a10bb904d89d8aa5b67498d44245f4a38827b07af44e82d94"
# Independent inventory and goldens retained from the reviewed release, not
# membership inferred from a submitted lock, receipt, registration or directory.
SKILLS = {
    "engineering/code-change-verification": ("9ac96d168f42c750d68cb1f860d74c94894a6a6faf41d814bfa76f5b57a47707", "0e034ef1ad1c1eb0a15dc7b7ba15ecfdc6931fe5267c38a4e0a4bf354fb04934"),
    "engineering/coverage-gap-reporter": ("cf2739728a0e9ce696b8d13a52621e6215093387eaecc9eee0affccad92c80d3", "1e1949795c2be3ba166cad87c230276e8a23fc48cec9639a369d62747c532682"),
    "engineering/implementation-strategy": ("8d24a55aff19cafa785615732bd935e62543ce9416d9602232c4ecaeba875ce3", "695b5bccd308e33ab968c5673c3fc37b84ad7d18af13a85e141e3dc627cda6b0"),
    "engineering/pr-review-and-draft": ("8fa20412346854320e87ac3fbad67212fb9abb2dd5e8a74d8445e9fbc52cc0c3", "b88de88b4f28d615793f89babad9231ae779f7695e4eae33601e2363375626d5"),
    "engineering/test-gap-analyzer": ("5f884dd4825f3dc7c5487ba5df70a5822ad4cd0ba521be7b36055e512b310a86", "67dd67f56770a7aa04411691bf5d589fe21295daba0b4762be5b8ef6c5f1d0dd"),
}
ROOT_FILES = {"README.md", "plugin.yaml", "plugin.json", "dependencies.lock.json", "checksums.txt", "sbom.cdx.json",
              "provenance.json", "scripts/first_use.mjs", "examples/first-use-input.json", "examples/overview.md",
              "hooks/verification-before-complete.md", "tests/test-prompts.md"}


def require(condition, message):
    if not condition:
        raise ValueError(message)


def sha(data):
    return hashlib.sha256(data).hexdigest()


def inventory(files):
    names = set(files)
    for name in files:
        names.update(p.as_posix() for p in PurePosixPath(name).parents if p != PurePosixPath("."))
    return names


def component(files, id):
    prefix = f"bundled/skills/{id}/"
    return {name[len(prefix):]: data for name, data in files.items() if name.startswith(prefix)}


def verify_archive(files, modes):
    require(not any(PurePosixPath(name).name == RECEIPT for name in files), "release contains an install receipt")
    require(set(modes) == inventory(files), "archive contains unexpected empty directories")
    root_files = {name for name in files if not name.startswith("bundled/")}
    require(root_files == ROOT_FILES, "plugin root inventory differs from reviewed release")
    lock = json.loads(files["dependencies.lock.json"])
    require(lock.get("lock_version") == "1.0" and lock.get("root") == {"module": "plugins", "id": ID, "version": "0.2.0"},
            "plugin lock root differs from pinned release")
    members = lock.get("components", [])
    require(len(members) == 5 and {m.get("id") for m in members} == set(SKILLS), "lock must contain all five pinned skills")
    allowed = set(ROOT_FILES)
    for member in members:
        id = member["id"]
        expected = {"module": "skills", "id": id, "version": "0.1.0", "content_sha256": SKILLS[id][0],
                    "manifest_path": f"bundled/skills/{id}/skill.yaml", "dependencies": []}
        require(member == expected, "lock component identity, path, edges or digest changed")
        content = component(files, id)
        require({"README.md", "SKILL.md", "skill.yaml"} <= content.keys(), "bundled skill is incomplete")
        prefix = f"bundled/skills/{id}/"
        lines = [f"{sha(data)} {modes[prefix+name]:o} {len(data)} {name}" for name, data in content.items()]
        require(sha(("\n".join(sorted(lines)) + "\n").encode()) == SKILLS[id][0], "bundled skill bytes/modes differ from reviewed closure")
        allowed.update(prefix + name for name in content)
    require(set(files) == allowed, "archive contains undeclared bundled components")
    checksums = {}
    for line in files["checksums.txt"].decode().splitlines():
        digest, name = line.split("  ", 1)
        require(name not in checksums and len(digest) == 64, "duplicate or invalid artifact checksum")
        checksums[name] = digest
    require(set(checksums) == set(files) - {"checksums.txt"}, "artifact checksum inventory is incomplete")
    require(all(sha(files[name]) == digest for name, digest in checksums.items()), "artifact checksum mismatch")


def read_tree(root):
    require(root.is_dir() and not root.is_symlink(), "installed package is missing or linked")
    files, modes = {}, {}
    for path in root.rglob("*"):
        info = path.lstat()
        name = path.relative_to(root).as_posix()
        require(stat.S_ISDIR(info.st_mode) or stat.S_ISREG(info.st_mode), "installed tree contains a link or special file")
        modes[name] = stat.S_IMODE(info.st_mode)
        if stat.S_ISREG(info.st_mode):
            require(info.st_nlink == 1, "installed file has an external hardlink")
            files[name] = path.read_bytes()
    return files, modes


def tree_digest(files, modes):
    digest = hashlib.sha256()
    for name in sorted(modes):
        if name == RECEIPT:
            continue
        kind = "file" if name in files else "dir"
        mode = 0o755 if kind == "dir" or modes[name] & 0o111 else 0o644
        digest.update(f"{kind}\0{name}\0{mode:o}\0".encode())
        if kind == "file":
            digest.update(files[name])
        digest.update(b"\0")
    return digest.hexdigest()


def receipt(document, files, modes, id, runtime, artifact):
    bundled = id != ID
    expected = {"schema_version": "skills-hub.install-receipt/v1", "source": "bundled" if bundled else "remote",
                "module": "skills" if bundled else "plugins", "id": id, "version": "0.1.0" if bundled else "0.2.0",
                "runtime": runtime, "runtime_contract_sha256": SKILLS[id][1] if bundled else ROOT_CONTRACT,
                "tree_sha256": tree_digest(files, modes), "parent_artifact_sha256" if bundled else "artifact_sha256": artifact}
    require(set(document) == set(expected) | {"installed_at"} | (set() if bundled else {"closure"}), "receipt fields differ from pinned role")
    require(all(document.get(key) == value for key, value in expected.items()), "receipt identity/integrity differs from installed closure")
    instant = datetime.datetime.fromisoformat(document["installed_at"].replace("Z", "+00:00"))
    require(instant.year > 1 and instant.tzinfo is not None, "receipt install timestamp is invalid")


def expected_registration(runtime, pinned):
    return {"schema_version": "skills-hub.runtime-package/v1", "runtime": runtime, "source_lock": "dependencies.lock.json",
            "package": {"id": ID, "name": pinned["name"], "version": "0.2.0", "description": pinned["description"]},
            "registrations": {"skills": [{"id": id, "version": "0.1.0", "path": "./skills/" + id.split("/")[-1],
                                            "enforcement": "native"} for id in sorted(SKILLS)],
                              "agents": [], "mcp_servers": [], "config": [],
                              "hooks": [{"id": "verification-before-complete", "path": "./hooks/verification-before-complete.md",
                                         "enforcement": "advisory", "reason": "the source hook is guidance, not an executable native hook definition"}]}}


def verify_install(runtime_root, runtime, files, modes, artifact, pinned, inspector):
    root = runtime_root / "plugins" / ID
    actual, actual_modes = read_tree(root)
    expected = dict(files)
    expected[f".runtime/{runtime}.json"] = expected_registration(runtime, pinned)
    native = {"name": "code-maintenance-plugin", "version": "0.2.0", "description": pinned["description"]}
    if runtime == "codex":
        expected["plugin.json"] = {"$schema": "https://agent-plugins.org/schemas/1.0.0/plugin.schema.json", **native}
    elif runtime == "claude":
        expected[".claude-plugin/plugin.json"] = {**native, "skills": "./skills/"}
    members = []
    receipts = {RECEIPT}
    for id in sorted(SKILLS):
        source = component(files, id)
        for relative, data in source.items():
            expected[f"skills/{id.split('/')[-1]}/{relative}"] = data
        bundled_receipt = f"bundled/skills/{id}/{RECEIPT}"
        receipts.add(bundled_receipt)
        require(bundled_receipt in actual, "bundled skill receipt is missing")
        member_root = runtime_root / "skills" / id
        member_files, member_modes = read_tree(member_root)
        require(set(member_files) == set(source) | {RECEIPT}, "activated skill inventory differs from bundled skill")
        require(all(member_files[name] == data for name, data in source.items()), "activated skill bytes differ from archive")
        wanted_modes = {name: modes[f"bundled/skills/{id}/{name}"] for name in inventory(source)}
        require(member_modes == {**wanted_modes, RECEIPT: 0o644}, "activated skill permissions differ from archive")
        require(member_files[RECEIPT] == actual[bundled_receipt], "bundled and activated skill receipts differ")
        record = json.loads(member_files[RECEIPT])
        receipt(record, member_files, member_modes, id, runtime, artifact)
        members.append({"module": "skills", "id": id, "version": "0.1.0", "runtime_contract_sha256": SKILLS[id][1],
                        "tree_sha256": record["tree_sha256"]})
    require(set(actual) == set(expected) | receipts, "installed plugin inventory differs from archive and registered outputs")
    wanted_modes = {name: modes.get(name, 0o644 if name in expected else 0o755) for name in inventory(expected)}
    wanted_modes.update({name: 0o644 for name in receipts})
    require(actual_modes == wanted_modes, "installed plugin permissions or directory inventory changed")
    for name, data in expected.items():
        require(actual[name] == data if isinstance(data, bytes) else json.loads(actual[name]) == data,
                f"installed plugin payload/registration differs: {name}")
    record = json.loads(actual[RECEIPT])
    receipt(record, actual, actual_modes, ID, runtime, artifact)
    require(record.get("closure") == members, "root receipt must bind every pinned activated skill")
    verified = inspector(root)
    require(verified.get("receipt_verified") is True and verified.get("closure_verified") is True,
            "runtime receipt/closure consumer did not verify installation")
    # Keep installer lock files outside the package snapshots; they may be touched
    # while serializing another attempt. No other activation paths are allowed.
    packages = [root, *(runtime_root / "skills" / id for id in SKILLS)]
    outside = inventory({p.relative_to(runtime_root).as_posix(): b"" for p in packages})
    locks = {str(p.parent.relative_to(runtime_root) / ("." + p.name + ".skills-hub.lock")) for p in packages}
    for path in runtime_root.rglob("*"):
        relative = path.relative_to(runtime_root).as_posix()
        if any(path == p or p in path.parents for p in packages):
            continue
        require(not path.is_symlink(), "activation root contains an unexpected symlink")
        require(relative in outside or relative in locks and path.is_file() and path.read_bytes() == b"",
                "activation root contains an undeclared package or output")
    return {"receipt_verified": True, "closure_verified": True, "dependency_count": 5,
            "registration_verified": True, "hook_enforcement": "advisory",
            "payload_sha256": sha(json.dumps({k: sha(v) for k, v in sorted(actual.items())
                                             if PurePosixPath(k).name != RECEIPT}).encode())}


def closure_snapshot(runtime_root, snapshot):
    return {"plugins/" + ID: snapshot(runtime_root / "plugins" / ID),
            **{"skills/" + id: snapshot(runtime_root / "skills" / id) for id in SKILLS}}


def first_use(root, files, work, runner):
    require(b"node scripts/first_use.mjs examples/first-use-input.json" in files["README.md"], "published first-use guidance changed")
    node, version = runner.node_runtime()
    marker, guard = runner.network_guard(work)
    inputs = [root / "examples/first-use-input.json", work / "controlled-first-use.json"]
    sentinel = work / "verification-command-was-executed"
    inputs[1].write_text(json.dumps({"change": "Verification probe only", "risk": "No changes or merge authority",
                                   "verification_command": f"touch '{sentinel}'"}))
    for input_path in inputs:
        supplied = json.loads(input_path.read_text())
        with runner.Process([node, "--import", str(guard), str(root / "scripts/first_use.mjs"), str(input_path)], work) as process:
            output = process.document()
        expected = {"schema_version": "local-plugin.first-use/v1", "plugin": ID,
                    "plan": ["inspect affected boundary", "implement the smallest change", supplied["verification_command"], "report residual risk"],
                    "change": supplied["change"], "risk": supplied["risk"], "merge_authorized": False}
        require(output == expected and output.get("merge_authorized") is False, "first-use plan or merge boundary changed")
        require(not sentinel.exists(), "first-use helper executed the proposed verification command")
        require(not marker.exists(), "first-use helper attempted network access")
    return {"status": "passed", "probes": 2, "node_version": version, "network_attempts": 0,
            "merge_authorized": False, "verification_command_executed": False}


def verify_runtime(binary, registry_url, runtime, directory, files, modes, artifact, result, inspector, snapshot, runner):
    work = directory / runtime
    work.mkdir()
    cache = work / "cache"
    root = work / "runtime"
    pinned = inspector(None)
    def args(target):
        return [str(binary), "install", "--source", "remote", "--module", "plugins", "--registry-url", registry_url,
                "--entry", ID + "@0.2.0", "--runtime", runtime, "--target", str(target / "plugins"),
                "--execution-runtime", "node22", "--cache-dir", str(cache)]
    def run(arguments):
        return subprocess.run(arguments, cwd=work, env={"PATH": os.defpath}, capture_output=True, timeout=150)
    installed = run(args(root))
    require(installed.returncode == 0, "cold plugin install failed: " + installed.stderr.decode(errors="replace")[-2000:])
    result.update(verify_install(root, runtime, files, modes, artifact, pinned, inspector))
    result["cold_install"] = "passed"
    before = closure_snapshot(root, snapshot)
    result["first_use"] = first_use(root / "plugins" / ID, files, work, runner)
    require(closure_snapshot(root, snapshot) == before, "first-use mutated installed closure")
    verify_install(root, runtime, files, modes, artifact, pinned, inspector)
    duplicate = run(args(root) + ["--offline"])
    require(duplicate.returncode != 0 and b"already exists" in duplicate.stderr
            and closure_snapshot(root, snapshot) == before, "duplicate plugin install did not reject without closure mutation")
    result["duplicate_rejected_without_mutation"] = "passed"
    force = run(args(root) + ["--offline", "--force"])
    require(force.returncode != 0 and b"in-place" in force.stderr and b"closure" in force.stderr and b"not supported" in force.stderr
            and closure_snapshot(root, snapshot) == before, "in-place plugin force install did not preserve closure")
    result["in_place_force_rejected_without_mutation"] = "passed"
    fresh = work / "offline-runtime"
    with socket.socket() as blocked:
        blocked.bind(("127.0.0.1", 0))
        proxy = f"http://127.0.0.1:{blocked.getsockname()[1]}"
        offline = subprocess.run(args(fresh) + ["--offline"], cwd=work, capture_output=True, timeout=150,
                                 env={"PATH": os.defpath, "HTTP_PROXY": proxy, "HTTPS_PROXY": proxy, "NO_PROXY": ""})
    require(offline.returncode == 0, "offline clean plugin install failed: " + offline.stderr.decode(errors="replace")[-2000:])
    verify_install(fresh, runtime, files, modes, artifact, pinned, inspector)
    require(closure_snapshot(root, snapshot) == before, "offline clean install mutated original activation")
    result["offline_clean_install"] = "passed"
