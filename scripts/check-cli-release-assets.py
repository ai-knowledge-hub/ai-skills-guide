#!/usr/bin/env python3
"""Read-only publication guard: require a tag with no existing release record."""

import argparse
import json
import os
import re
import subprocess
import sys


def check_release_assets(repository, tag):
    if not re.fullmatch(r"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+", repository):
        raise ValueError("a valid GITHUB_REPOSITORY is required")
    if not re.fullmatch(r"v\d+\.\d+\.\d+(?:-[0-9A-Za-z.-]+)?", tag):
        raise ValueError("an exact release tag is required")
    if not os.environ.get("GH_TOKEN"):
        raise ValueError("GH_TOKEN is required to inspect draft and published releases")

    # The list endpoint includes drafts for the workflow's contents:write token.
    # Scan every page: a tag lookup that returns 404 must not disguise a draft,
    # permission problem or missing repository as a safe empty release.
    response = subprocess.run([
        "gh", "api", "--hostname", "github.com", "--method", "GET",
        "--paginate", "--slurp", f"repos/{repository}/releases?per_page=100",
    ], check=True, capture_output=True, text=True, timeout=60)
    pages = json.loads(response.stdout)
    if not isinstance(pages, list) or not pages or any(not isinstance(page, list) for page in pages):
        raise ValueError("release API returned an invalid page inventory")
    releases = [release for page in pages for release in page]
    if any(not isinstance(release, dict) or not isinstance(release.get("tag_name"), str)
           for release in releases):
        raise ValueError("release API returned invalid release metadata")
    matching = [release for release in releases if release["tag_name"] == tag]
    if len(matching) > 1:
        raise ValueError("release API returned ambiguous tag metadata")
    if matching:
        assets = matching[0].get("assets")
        if not isinstance(assets, list):
            raise ValueError("release API did not supply an asset inventory")
        if assets:
            raise ValueError(f"{tag} already has release assets; use a new version instead of republishing")
        raise ValueError(f"{tag} already has a release record; publication requires a new version")


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("tag")
    args = parser.parse_args()
    try:
        check_release_assets(os.environ.get("GITHUB_REPOSITORY", ""), args.tag)
    except (OSError, ValueError, subprocess.SubprocessError) as error:
        # Do not echo gh stderr, which may contain unrelated account details.
        message = str(error) if isinstance(error, ValueError) else "release API lookup failed"
        print(f"Refusing CLI publication: {message}", file=sys.stderr)
        return 1
    print(f"No existing release record for {args.tag}; draft creation may proceed")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
