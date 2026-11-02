#!/usr/bin/env python3
"""Publish a fresh draft only after all create-only asset uploads succeed."""

import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import sys
import urllib.error
import urllib.parse
import urllib.request


def require(condition, message):
    if not condition:
        raise ValueError(message)


class NoRedirects(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise ValueError("release API redirects are not permitted")


class GitHubAPI:
    def __init__(self, token):
        require(bool(token), "GH_TOKEN is required")
        self.token = token
        self.opener = urllib.request.build_opener(NoRedirects())

    def request(self, method, url, expected_status, document=None, payload=None):
        require(urllib.parse.urlsplit(url).netloc in {"api.github.com", "uploads.github.com"}
                and url.startswith("https://"), "unexpected release API origin")
        data = json.dumps(document).encode() if document is not None else payload
        request = urllib.request.Request(url, data=data, method=method, headers={
            "Authorization": f"Bearer {self.token}", "Accept": "application/vnd.github+json",
            "Content-Type": "application/json" if document is not None else "application/octet-stream",
            "User-Agent": "skills-hub-cli-release", "X-GitHub-Api-Version": "2022-11-28",
        })
        # No retry: an ambiguous write must stop with the draft retained.
        # In particular, upload HTTP 422 is an error, never a skip or overwrite.
        try:
            with self.opener.open(request, timeout=60) as response:
                require(response.status == expected_status, "unexpected release API status")
                data = response.read(2 * 1024 * 1024 + 1)
        except urllib.error.HTTPError as error:
            error.close()
            raise
        require(len(data) <= 2 * 1024 * 1024, "release API response is too large")
        return json.loads(data)


def artifact_payloads(directory, tag):
    binaries = {f"skills-hub_{tag}_{system}_{arch}{'.exe' if system == 'windows' else ''}"
                for system in ["linux", "darwin", "windows"] for arch in ["amd64", "arm64"]}
    names = binaries | {"SHA256SUMS", f"skills-hub_{tag}_provenance.json"}
    require(directory.is_dir() and {path.name for path in directory.iterdir()} == names,
            "release directory must contain exactly six binaries, checksums and provenance")
    payloads = {}
    for name in sorted(names):
        path = directory / name
        require(path.is_file() and not path.is_symlink(), "release assets must be regular files")
        payloads[name] = path.read_bytes()
        require(bool(payloads[name]), "release assets must not be empty")
    expected = {name: hashlib.sha256(payloads[name]).hexdigest() for name in binaries}
    rows = [line.split() for line in payloads["SHA256SUMS"].decode().splitlines()]
    require(len(rows) == len(expected) and all(len(row) == 2 for row in rows), "invalid release checksums")
    require(len({row[1] for row in rows}) == len(rows)
            and {row[1]: row[0] for row in rows} == expected, "release checksum inventory mismatch")
    return payloads


def verify_release(release, release_id, tag, draft):
    require(isinstance(release, dict) and type(release.get("id")) is int
            and release["id"] == release_id and release.get("tag_name") == tag
            and release.get("draft") is draft, "release identity or draft state changed")


def asset_identity(asset, name, payload):
    require(isinstance(asset, dict) and type(asset.get("id")) is int and asset["id"] > 0,
            "upload did not return a valid asset identity")
    expected = (name, len(payload), "sha256:" + hashlib.sha256(payload).hexdigest(), "uploaded")
    actual = (asset.get("name"), asset.get("size"), asset.get("digest"), asset.get("state"))
    require(actual == expected, "uploaded asset name, bytes or state differ from the build")
    return asset["id"]


def verify_inventory(inventory, payloads, uploaded):
    # There must be exactly eight assets, so a full first page is already invalid.
    require(isinstance(inventory, list) and len(inventory) == len(payloads), "release asset inventory changed")
    seen = set()
    for asset in inventory:
        name = asset.get("name") if isinstance(asset, dict) else None
        require(name in uploaded and name not in seen, "unexpected or duplicate release asset")
        require(asset_identity(asset, name, payloads[name]) == uploaded[name], "release asset was replaced")
        seen.add(name)


def publish(api, repository, tag, commit, directory):
    require(re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repository), "invalid repository")
    require(re.fullmatch(r"v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?", tag), "invalid release tag")
    require(re.fullmatch(r"[a-f0-9]{40}", commit), "an exact source commit is required")
    payloads = artifact_payloads(directory, tag)
    base = f"https://api.github.com/repos/{repository}/releases"
    # Never look up and reuse an existing release, even an empty draft.
    release = api.request("POST", base, 201, document={
        "tag_name": tag, "target_commitish": commit, "draft": True,
        "generate_release_notes": True, "prerelease": "-" in tag,
    })
    release_id = release.get("id") if isinstance(release, dict) else None
    require(type(release_id) is int and release_id > 0, "draft creation did not return a release identity")
    verify_release(release, release_id, tag, True)
    uploaded = {}
    for name, payload in payloads.items():
        url = f"https://uploads.github.com/repos/{repository}/releases/{release_id}/assets?name={urllib.parse.quote(name)}"
        asset = api.request("POST", url, 201, payload=payload)
        uploaded[name] = asset_identity(asset, name, payload)
    require(len(set(uploaded.values())) == len(uploaded), "upload returned duplicate asset identities")
    release_url = f"{base}/{release_id}"
    verify_release(api.request("GET", release_url, 200), release_id, tag, True)
    inventory = api.request("GET", f"{release_url}/assets?per_page=100", 200)
    verify_inventory(inventory, payloads, uploaded)
    result = api.request("PATCH", release_url, 200, document={"draft": False})
    verify_release(result, release_id, tag, False)
    verify_inventory(result.get("assets"), payloads, uploaded)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("tag")
    parser.add_argument("commit")
    parser.add_argument("directory", type=Path)
    args = parser.parse_args()
    try:
        publish(GitHubAPI(os.environ.get("GH_TOKEN", "")), os.environ.get("GITHUB_REPOSITORY", ""),
                args.tag, args.commit, args.directory)
    except (OSError, ValueError) as error:
        # Keep credentials and API bodies out of logs; do not delete or retry.
        message = str(error) if isinstance(error, ValueError) else "release API operation failed"
        print(f"CLI publication failed: {message}. Retain and inspect any draft; do not reuse its assets.", file=sys.stderr)
        return 1
    print(f"Published {args.tag} after eight verified asset uploads")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
