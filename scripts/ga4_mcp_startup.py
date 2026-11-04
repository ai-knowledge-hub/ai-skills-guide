"""Bounded process-only verification of the retained GA4 setup path.

This is not a provider/authentication check or an OS-wide network sandbox.
"""

import json
import os
from pathlib import Path
import re
import selectors
import shutil
import signal
import subprocess
import time


def require(condition, message):
    if not condition:
        raise ValueError(message)


class Process:
    """Bound elapsed time and combined output; reap the process group on exit."""

    def __init__(self, command, cwd, timeout=15, limit=256 * 1024):
        self.child = subprocess.Popen(command, cwd=cwd, env={"PATH": os.defpath},
                                      stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                      stderr=subprocess.PIPE, start_new_session=True)
        self.deadline, self.limit = time.monotonic() + timeout, limit
        self.output, self.used = b"", 0
        self.selector = selectors.DefaultSelector()
        for stream in (self.child.stdout, self.child.stderr):
            os.set_blocking(stream.fileno(), False)
            self.selector.register(stream, selectors.EVENT_READ)

    def __enter__(self):
        return self

    def __exit__(self, *_):
        try:
            if self.child.poll() is None:
                try:
                    os.killpg(self.child.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
                except PermissionError:
                    self.child.kill()
            self.child.wait(timeout=5)
        finally:
            self.selector.close()
            for stream in (self.child.stdin, self.child.stdout, self.child.stderr):
                try:
                    stream.close()
                except BrokenPipeError:
                    # Preserve the original startup failure and still close the
                    # other descriptors if the server exited before reading.
                    pass

    def pump(self):
        remaining = self.deadline - time.monotonic()
        require(remaining > 0, "GA4 process timed out")
        events = self.selector.select(remaining)
        require(events, "GA4 process timed out")
        for key, _ in events:
            data = os.read(key.fileobj.fileno(), 65536)
            if not data:
                self.selector.unregister(key.fileobj)
                continue
            self.used += len(data)
            require(self.used <= self.limit, "GA4 process output exceeded the limit")
            if key.fileobj is self.child.stdout:
                self.output += data

    def send(self, message):
        try:
            self.child.stdin.write(json.dumps(message).encode() + b"\n")
            self.child.stdin.flush()
        except BrokenPipeError as error:
            raise ValueError("GA4 process closed input before completing MCP startup") from error

    def read(self):
        while b"\n" not in self.output:
            require(self.selector.get_map(), "GA4 process exited without a complete response")
            self.pump()
        line, self.output = self.output.split(b"\n", 1)
        return json.loads(line)

    def finish(self):
        self.child.stdin.close()
        while self.selector.get_map():
            self.pump()
        remaining = self.deadline - time.monotonic()
        require(remaining > 0, "GA4 process timed out")
        require(self.child.wait(timeout=remaining) == 0, "GA4 process exited unsuccessfully")
        require(not self.output, "GA4 process emitted unexpected extra output")


def registration(guide, package_dir):
    blocks = re.findall(r"```json\n(.*?)\n```", guide, re.S)
    require(len(blocks) == 1, "setup guide must contain one unambiguous registration")
    settings = json.loads(blocks[0].replace("<absolute-installed-ga4-directory>", str(package_dir)))
    expected = {"type": "stdio", "command": "node",
                "args": ["--preserve-symlinks-main", str(package_dir / "scripts/ga4_mcp_server.mjs")]}
    require(settings == {"mcpServers": {"ga4": expected}}, "setup guide changed the pinned GA4 registration")
    return expected


def response(document, request_id):
    require(isinstance(document, dict) and document.get("jsonrpc") == "2.0"
            and type(document.get("id")) is int and document["id"] == request_id
            and "error" not in document and isinstance(document.get("result"), dict),
            "invalid GA4 MCP response envelope")
    return document["result"]


def verify(binary, install_output, package_dir, work):
    guidance = re.search(rb"^install.setup_guidance: (.+)$", install_output, re.M)
    require(guidance is not None and guidance.group(1).split() == [b"skills-hub", b"help", b"setup"],
            "installed GA4 package lacks the compatible setup redirect")
    guide = subprocess.run([str(binary), "help", "setup"], cwd=work, env={"PATH": os.defpath},
                           capture_output=True, check=True, timeout=15).stdout.decode()
    # Exercise the guide's symlink-safe absolute registration outside the package.
    linked = work / "installed-ga4-link"
    linked.symlink_to(package_dir, target_is_directory=True)
    server = registration(guide, linked)
    node = shutil.which("node")
    require(node is not None, "GA4 startup verification requires Node.js 22+")
    node = str(Path(node).resolve())
    version = subprocess.run([node, "--version"], env={"PATH": os.defpath},
                             capture_output=True, check=True, timeout=10).stdout.decode().strip()
    require(re.fullmatch(r"v\d+\.\d+\.\d+", version) and int(version[1:].split(".")[0]) >= 22,
            "GA4 startup verification requires Node.js 22+")
    marker = work / "network-attempts"
    guard = work / "block-network.mjs"
    # Catch attempted provider calls even if the package catches the exception.
    # This covers the retained server's fetch path and ordinary Node socket APIs;
    # it does not claim to contain arbitrary native code or spawned subprocesses.
    guard.write_text("""import fs from 'node:fs';
import net from 'node:net';
import http from 'node:http';
import https from 'node:https';
import tls from 'node:tls';
import {syncBuiltinESMExports} from 'node:module';
const deny = () => {
  fs.appendFileSync(MARKER, 'attempt\\n');
  throw new Error('network forbidden in credential-free startup');
};
globalThis.fetch = deny;
net.Socket.prototype.connect = deny;
net.connect = net.createConnection = deny;
http.request = http.get = https.request = https.get = tls.connect = deny;
syncBuiltinESMExports();
""".replace("MARKER", json.dumps(str(marker))))
    command = [node, "--import", str(guard), *server["args"]]
    with Process(command + ["--healthcheck"], work) as process:
        health = process.read()
        process.finish()
    require(health == {"status": "ok", "server": "ga4-mcp-connector", "version": "0.2.0", "transport": "stdio"},
            "GA4 healthcheck did not identify the pinned server")
    require(not marker.exists(), "GA4 startup attempted network access")
    with Process(command, work) as process:
        process.send({"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {
            "protocolVersion": "2025-06-18", "capabilities": {},
            "clientInfo": {"name": "skills-hub-published-check", "version": "1"}}})
        initialized = response(process.read(), 1)
        require(initialized.get("protocolVersion") == "2025-06-18"
                and initialized.get("serverInfo") == {"name": "ga4-mcp-connector", "version": "0.2.0"}
                and initialized.get("capabilities", {}).get("tools") == {"listChanged": False},
                "GA4 initialize did not identify the pinned MCP contract")
        process.send({"jsonrpc": "2.0", "method": "notifications/initialized", "params": {}})
        process.send({"jsonrpc": "2.0", "id": 2, "method": "tools/list", "params": {}})
        tools = response(process.read(), 2).get("tools")
        require(isinstance(tools, list) and len(tools) == 1 and isinstance(tools[0], dict)
                and tools[0].get("name") == "ga4_run_report",
                "GA4 tools/list did not expose the pinned tool")
        schema = tools[0].get("inputSchema", {})
        require(schema.get("type") == "object" and {"property_id", "start_date", "end_date", "metrics"}
                <= set(schema.get("required", [])) and schema.get("additionalProperties") is False,
                "GA4 tool input contract is incomplete")
        properties = schema.get("properties", {})
        require(all(properties.get(name, {}).get("type") == "string" for name in ["property_id", "start_date", "end_date"])
                and properties.get("metrics", {}).get("type") == "array"
                and properties["metrics"].get("minItems") == 1, "GA4 tool input properties are incomplete")
        process.finish()
    require(not marker.exists(), "GA4 startup attempted network access")
    return {"setup_guidance": "passed", "healthcheck": "passed", "mcp_initialize": "passed",
            "mcp_tools_list": "passed", "network_attempts": 0, "node_version": version,
            "credentials_supplied": False, "provider_calls_verified": False}
