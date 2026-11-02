"""Release authority comes from an exact tag on main, never branch names alone."""

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
import urllib.error
import urllib.parse
from unittest.mock import patch


ROOT = Path(__file__).resolve().parents[2]
VERIFY = ROOT / "scripts/verify-cli-release-source.sh"
DISPATCH = ROOT / "scripts/dispatch-cli-release.sh"
TAG = "v0.4.0-alpha.1"
spec = importlib.util.spec_from_file_location("release_assets", ROOT / "scripts/check-cli-release-assets.py")
assets = importlib.util.module_from_spec(spec)
spec.loader.exec_module(assets)
spec = importlib.util.spec_from_file_location("publisher", ROOT / "scripts/publish-cli-release.py")
publisher = importlib.util.module_from_spec(spec)
spec.loader.exec_module(publisher)


class ReleaseAPIModel:
    """Independent server model: duplicate upload names produce HTTP 422."""

    def __init__(self, collision=None, mutation=None, fail_create=False):
        self.collision, self.mutation, self.fail_create = collision, mutation, fail_create
        self.release = {"id": 91, "tag_name": TAG, "draft": True}
        self.assets, self.payloads, self.calls, self.attempts = {}, {}, [], []
        self.published = False

    def request(self, method, url, expected_status, document=None, payload=None):
        self.calls.append((method, url, copy.deepcopy(document)))
        parsed = urllib.parse.urlsplit(url)
        if method == "POST" and parsed.path.endswith("/releases"):
            if self.fail_create:
                with contextlib.closing(urllib.error.HTTPError(url, 422, "release exists", {}, io.BytesIO())) as error:
                    raise error
            if document["draft"] is not True:
                raise AssertionError("release must start as a draft")
            return copy.deepcopy(self.release)
        if method == "POST" and parsed.hostname == "uploads.github.com":
            name = urllib.parse.parse_qs(parsed.query)["name"][0]
            self.attempts.append(name)
            if name == self.collision:
                # External writer wins after the workflow's final empty check.
                self.assets[name] = {"id": 999, "name": name}
                self.payloads[name] = b"external bytes that must not be replaced"
            if name in self.assets:
                with contextlib.closing(urllib.error.HTTPError(url, 422, "asset already exists", {}, io.BytesIO())) as error:
                    raise error
            self.payloads[name] = payload
            self.assets[name] = {"id": 100 + len(self.assets), "name": name, "state": "uploaded",
                                 "size": len(payload), "digest": "sha256:" + hashlib.sha256(payload).hexdigest()}
            result = copy.deepcopy(self.assets[name])
            if self.mutation == "bad-upload-response":
                result["digest"] = "sha256:" + "0" * 64
            return result
        if method == "GET" and parsed.path.endswith("/assets"):
            result = copy.deepcopy(list(self.assets.values()))
            if self.mutation == "extra":
                result.append({"name": "external-extra", "id": 999})
            elif self.mutation == "missing":
                result.pop()
            elif self.mutation == "replaced":
                result[0]["id"] = 999
            elif self.mutation == "changed-bytes":
                result[0]["digest"] = "sha256:" + "0" * 64
            elif self.mutation == "renamed":
                result[0]["name"] = "renamed"
            return result
        if method == "GET":
            result = copy.deepcopy(self.release)
            if self.mutation == "wrong-release":
                result["id"] = 92
            elif self.mutation == "premature-publish":
                result["draft"] = False
            return result
        if method == "PATCH":
            if document != {"draft": False}:
                raise AssertionError("unexpected release mutation")
            self.published = True
            self.release["draft"] = False
            result = copy.deepcopy(self.release)
            result["assets"] = copy.deepcopy(list(self.assets.values()))
            if self.mutation == "publication-response-changed":
                result["assets"].append({"name": "external-extra", "id": 999})
            return result
        raise AssertionError(f"unexpected API operation: {method} {url}")


class CLIReleasePublisherTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="cli-publisher-test-")
        self.addCleanup(self.temporary.cleanup)
        self.directory = Path(self.temporary.name)
        # Explicit target contract, independent of the publisher's inventory.
        self.binaries = [f"skills-hub_{TAG}_{target}" for target in [
            "linux_amd64", "linux_arm64", "darwin_amd64", "darwin_arm64", "windows_amd64.exe", "windows_arm64.exe"]]
        for name in self.binaries:
            (self.directory / name).write_bytes(("fixture " + name).encode())
        sums = "".join(f"{hashlib.sha256((self.directory / name).read_bytes()).hexdigest()}  {name}\n"
                       for name in self.binaries)
        (self.directory / "SHA256SUMS").write_text(sums)
        (self.directory / f"skills-hub_{TAG}_provenance.json").write_text("{}\n")

    def publish(self, api):
        publisher.publish(api, "ai-knowledge-hub/ai-skills-guide", TAG, "a" * 40, self.directory)

    def test_publish_happens_only_after_eight_matching_owned_uploads(self):
        api = ReleaseAPIModel()
        self.publish(api)
        self.assertTrue(api.published)
        self.assertEqual(set(api.assets), {path.name for path in self.directory.iterdir()})
        self.assertEqual(api.calls[-1][0], "PATCH")
        self.assertEqual(len([call for call in api.calls if call[0] == "PATCH"]), 1)
        for name, payload in api.payloads.items():
            self.assertEqual(payload, (self.directory / name).read_bytes())

    def test_external_same_name_upload_at_each_position_fails_without_skip_replace_or_publish(self):
        for index, name in enumerate(sorted(path.name for path in self.directory.iterdir())):
            with self.subTest(name=name):
                api = ReleaseAPIModel(collision=name)
                with self.assertRaises(urllib.error.HTTPError) as error:
                    self.publish(api)
                self.assertEqual(error.exception.code, 422)
                self.assertEqual(len(api.attempts), index + 1, "no upload may follow a collision")
                self.assertEqual(api.payloads[name], b"external bytes that must not be replaced")
                self.assertFalse(api.published)
                self.assertTrue(api.release["draft"])
                self.assertFalse(any(call[0] in {"PATCH", "DELETE"} for call in api.calls))

    def test_release_created_by_another_writer_is_not_reused(self):
        api = ReleaseAPIModel(fail_create=True)
        with self.assertRaises(urllib.error.HTTPError):
            self.publish(api)
        self.assertEqual(len(api.calls), 1)
        self.assertFalse(api.published)

    def test_inventory_or_release_mutations_block_draft_publication(self):
        for mutation in ["extra", "missing", "replaced", "changed-bytes", "renamed", "bad-upload-response",
                         "wrong-release", "premature-publish"]:
            with self.subTest(mutation=mutation):
                api = ReleaseAPIModel(mutation=mutation)
                with self.assertRaises(ValueError):
                    self.publish(api)
                self.assertFalse(api.published)
                self.assertFalse(any(call[0] in {"PATCH", "DELETE"} for call in api.calls))

    def test_invalid_local_inventory_and_checksums_stop_before_any_write(self):
        for mutation in ["missing", "extra", "checksum", "link", "empty"]:
            with self.subTest(mutation=mutation), tempfile.TemporaryDirectory() as temporary:
                directory = Path(temporary) / "dist"
                shutil.copytree(self.directory, directory)
                sample = directory / self.binaries[0]
                if mutation == "missing":
                    sample.unlink()
                elif mutation == "extra":
                    (directory / "extra").write_text("unexpected")
                elif mutation == "checksum":
                    sample.write_text("substituted")
                elif mutation == "empty":
                    sample.write_bytes(b"")
                else:
                    sample.unlink()
                    sample.symlink_to(self.directory / self.binaries[0])
                api = ReleaseAPIModel()
                with self.assertRaises(ValueError):
                    publisher.publish(api, "ai-knowledge-hub/ai-skills-guide", TAG, "a" * 40, directory)
                self.assertFalse(api.calls)

    def test_ambiguous_publication_response_is_failure_not_success_or_retry(self):
        api = ReleaseAPIModel(mutation="publication-response-changed")
        with self.assertRaisesRegex(ValueError, "inventory changed"):
            self.publish(api)
        self.assertEqual(len([call for call in api.calls if call[0] == "PATCH"]), 1)
        self.assertFalse(any(call[0] == "DELETE" for call in api.calls))

    def test_http_conflict_is_propagated_without_retry(self):
        api = publisher.GitHubAPI("fixture-not-a-credential")
        conflict = urllib.error.HTTPError("https://uploads.github.com/fixture", 422, "conflict", {}, io.BytesIO())
        with patch.object(api.opener, "open", side_effect=conflict) as request:
            with self.assertRaises(urllib.error.HTTPError):
                api.request("POST", "https://uploads.github.com/fixture", 201, payload=b"fixture")
        request.assert_called_once()

    def test_raw_upload_requires_created_status_and_preserves_request_bytes(self):
        for status in [201, 200]:
            with self.subTest(status=status):
                api = publisher.GitHubAPI("fixture-not-a-credential")
                response = io.BytesIO(b'{"id": 101}')
                response.status = status
                with patch.object(api.opener, "open", return_value=response) as send:
                    if status == 201:
                        self.assertEqual(api.request("POST", "https://uploads.github.com/fixture", 201,
                                                     payload=b"exact binary bytes"), {"id": 101})
                    else:
                        with self.assertRaises(ValueError):
                            api.request("POST", "https://uploads.github.com/fixture", 201, payload=b"exact binary bytes")
                request = send.call_args.args[0]
                self.assertEqual(request.get_method(), "POST")
                self.assertEqual(request.data, b"exact binary bytes")
                self.assertEqual(request.get_header("Content-type"), "application/octet-stream")

    def test_credentials_never_follow_redirects_or_unexpected_upload_hosts(self):
        api = publisher.GitHubAPI("fixture-not-a-credential")
        with patch.object(api.opener, "open") as request:
            with self.assertRaises(ValueError):
                api.request("POST", "https://external.example/upload", 201, payload=b"fixture")
            request.assert_not_called()
        with self.assertRaises(ValueError):
            publisher.NoRedirects().redirect_request(None, None, 302, "redirect", {}, "https://external.example")

    def test_upload_failure_returns_nonzero_and_never_reports_publication_success(self):
        with patch("sys.argv", ["publish", TAG, "a" * 40, str(self.directory)]), \
                patch.dict(os.environ, {"GH_TOKEN": "fixture-not-a-credential"}), \
                patch.object(publisher, "publish", side_effect=urllib.error.HTTPError(
                    "https://uploads.github.com/fixture", 422, "conflict", {}, io.BytesIO())), \
                contextlib.redirect_stderr(io.StringIO()) as stderr, contextlib.redirect_stdout(io.StringIO()) as stdout:
            self.assertEqual(publisher.main(), 1)
        self.assertIn("publication failed", stderr.getvalue())
        self.assertEqual(stdout.getvalue(), "")


class CLIReleaseAssetTests(unittest.TestCase):
    def query(self, pages):
        response = subprocess.CompletedProcess([], 0, json.dumps(pages), "")
        with patch.dict(os.environ, {"GH_TOKEN": "fixture-not-a-credential"}), \
                patch.object(assets.subprocess, "run", return_value=response) as run:
            assets.check_release_assets("ai-knowledge-hub/ai-skills-guide", TAG)
        return run

    def test_new_tag_allows_publication(self):
        for pages in [[[]],
                      [[{"tag_name": "v0.3.0", "assets": [{"name": "old-binary"}]}]]]:
            with self.subTest(pages=pages):
                run = self.query(pages)
                args = run.call_args.args[0]
                self.assertIn("--paginate", args)
                self.assertIn("--slurp", args)
                self.assertIn("repos/ai-knowledge-hub/ai-skills-guide/releases?per_page=100", args)
                self.assertTrue(run.call_args.kwargs["check"])

    def test_empty_existing_releases_are_not_adopted(self):
        for draft in [False, True]:
            with self.subTest(draft=draft), self.assertRaisesRegex(ValueError, "already has a release record"):
                self.query([[{"tag_name": TAG, "draft": draft, "assets": []}]])

    def test_any_existing_asset_blocks_including_drafts_partial_uploads_and_later_pages(self):
        for draft in [False, True]:
            for name in ["binary", "SHA256SUMS", "provenance.json", "unrelated-asset"]:
                for pages in [1, 2]:
                    with self.subTest(draft=draft, name=name, pages=pages):
                        release = {"tag_name": TAG, "draft": draft, "assets": [{"name": name}]}
                        inventory = [[release]] if pages == 1 else [[{"tag_name": "v9.0.0"}], [release]]
                        with self.assertRaisesRegex(ValueError, "already has release assets"):
                            self.query(inventory)

    def test_new_assets_between_checks_block_the_final_check(self):
        self.query([[]])
        with self.assertRaisesRegex(ValueError, "already has release assets"):
            self.query([[{"tag_name": TAG, "assets": [{"name": "binary"}]}]])

    def test_uncertain_inventory_never_authorizes_publication(self):
        cases = [[], {}, [None], [[None]], [[{}]], [[{"tag_name": TAG}]],
                 [[{"tag_name": TAG, "assets": None}]], [[{"tag_name": TAG, "assets": {}}]],
                 [[{"tag_name": TAG, "assets": []}], [{"tag_name": TAG, "assets": []}]]]
        for pages in cases:
            with self.subTest(pages=pages), self.assertRaises(ValueError):
                self.query(pages)

    def test_api_auth_rate_limit_not_found_timeout_and_malformed_responses_fail_closed(self):
        for status in [401, 403, 404, 429, 500]:
            with self.subTest(status=status), patch.dict(os.environ, {"GH_TOKEN": "fixture-not-a-credential"}), \
                    patch.object(assets.subprocess, "run", side_effect=subprocess.CalledProcessError(
                        1, ["gh", "api"], stderr=f"HTTP {status}")), self.assertRaises(subprocess.SubprocessError):
                assets.check_release_assets("ai-knowledge-hub/ai-skills-guide", TAG)
        for result in [subprocess.TimeoutExpired(["gh"], 60), FileNotFoundError("gh missing")]:
            with self.subTest(result=result), patch.dict(os.environ, {"GH_TOKEN": "fixture-not-a-credential"}), \
                    patch.object(assets.subprocess, "run", side_effect=result), \
                    self.assertRaises((OSError, subprocess.SubprocessError)):
                assets.check_release_assets("ai-knowledge-hub/ai-skills-guide", TAG)
        with patch.dict(os.environ, {"GH_TOKEN": "fixture-not-a-credential"}), \
                patch.object(assets.subprocess, "run", return_value=subprocess.CompletedProcess([], 0, "bad JSON", "")), \
                self.assertRaises(ValueError):
            assets.check_release_assets("ai-knowledge-hub/ai-skills-guide", TAG)

    def test_missing_token_or_invalid_identity_stops_before_network(self):
        for repository, tag, token in [("", TAG, "fixture"), ("owner/repo/other", TAG, "fixture"),
                                       ("owner/repo", "main", "fixture"), ("owner/repo", TAG, "")]:
            with self.subTest(repository=repository, tag=tag), patch.dict(os.environ, {"GH_TOKEN": token}), \
                    patch.object(assets.subprocess, "run") as run, self.assertRaises(ValueError):
                assets.check_release_assets(repository, tag)
            run.assert_not_called()

    def test_lookup_error_returns_nonzero_without_echoing_cli_stderr(self):
        with patch("sys.argv", ["check", TAG]), \
                patch.object(assets, "check_release_assets", side_effect=subprocess.CalledProcessError(
                    1, ["gh"], stderr="private account details")), contextlib.redirect_stderr(io.StringIO()) as stderr:
            self.assertEqual(assets.main(), 1)
        self.assertIn("Refusing CLI publication", stderr.getvalue())
        self.assertNotIn("private account details", stderr.getvalue())


class CLIReleaseTests(unittest.TestCase):
    def setUp(self):
        self.temporary = tempfile.TemporaryDirectory(prefix="cli-release-test-")
        self.addCleanup(self.temporary.cleanup)
        self.directory = Path(self.temporary.name)
        self.repo = self.directory / "repo"
        self.repo.mkdir()
        self.git("init", "-q", "-b", "main")
        self.git("config", "user.name", "Release fixture")
        self.git("config", "user.email", "release@example.invalid")
        self.git("config", "commit.gpgsign", "false")
        (self.repo / "payload").write_text("reviewed source\n")
        self.git("add", "payload")
        self.git("commit", "-qm", "reviewed source")
        self.commit = self.git("rev-parse", "HEAD")
        self.git("update-ref", "refs/remotes/origin/main", self.commit)
        self.git("tag", TAG)
        self.bin = self.directory / "bin"
        self.bin.mkdir()
        self.calls = self.directory / "dispatch-arguments"
        fake_gh = self.bin / "gh"
        fake_gh.write_text('#!/bin/sh\nprintf "%s\\n" "$@" > "$RELEASE_TEST_CALLS"\nexit "${RELEASE_TEST_EXIT:-0}"\n')
        fake_gh.chmod(0o755)

    def git(self, *args):
        return subprocess.check_output(["git", *args], cwd=self.repo, stderr=subprocess.PIPE, text=True).strip()

    def run_script(self, script, *args, **environment):
        return subprocess.run(["bash", str(script), *args], cwd=self.repo, capture_output=True, text=True,
                              env={**os.environ, "PATH": f"{self.bin}{os.pathsep}{os.environ['PATH']}",
                                   "GITHUB_REPOSITORY": "ai-knowledge-hub/ai-skills-guide",
                                   "RELEASE_TEST_CALLS": str(self.calls), **environment})

    def test_exact_lightweight_and_annotated_tags_on_integrated_source(self):
        for annotated in [False, True]:
            with self.subTest(annotated=annotated):
                if annotated:
                    self.git("tag", "-d", TAG)
                    self.git("tag", "-a", TAG, "-m", "annotated release")
                result = self.run_script(VERIFY, TAG, self.commit)
                self.assertEqual(result.returncode, 0, result.stderr)

    def test_missing_malformed_or_branch_only_ref_cannot_authorize_release(self):
        self.git("branch", "v1.2.3")
        for tag in ["", "main", "v1", "v1.2.3", "v0.4.0-alpha.1/other", "$(touch injected)"]:
            with self.subTest(tag=tag):
                self.assertNotEqual(self.run_script(VERIFY, tag).returncode, 0)
        self.assertFalse((self.repo / "injected").exists())

    def test_valid_tag_on_unmerged_feature_source_is_rejected(self):
        self.git("switch", "-qc", "feature")
        (self.repo / "payload").write_text("unreviewed source\n")
        self.git("commit", "-qam", "feature source")
        self.git("tag", "-f", TAG)
        result = self.run_script(VERIFY, TAG)
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("origin/main", result.stderr)

    def test_tag_expected_source_and_checkout_cannot_be_substituted(self):
        self.assertNotEqual(self.run_script(VERIFY, TAG, "0" * 40).returncode, 0)
        (self.repo / "payload").write_text("another integrated commit\n")
        self.git("commit", "-qam", "another commit")
        self.git("update-ref", "refs/remotes/origin/main", self.git("rev-parse", "HEAD"))
        self.assertNotEqual(self.run_script(VERIFY, TAG, self.commit).returncode, 0)

    def test_missing_main_authority_or_dirty_checkout_is_rejected(self):
        for mutation in ["missing-main", "tracked-change", "untracked-change"]:
            with self.subTest(mutation=mutation):
                self.git("update-ref", "refs/remotes/origin/main", self.commit)
                self.git("checkout", "--", "payload")
                if mutation == "missing-main":
                    self.git("update-ref", "-d", "refs/remotes/origin/main")
                elif mutation == "tracked-change":
                    (self.repo / "payload").write_text("uncommitted\n")
                else:
                    (self.repo / "extra").write_text("untracked\n")
                self.assertNotEqual(self.run_script(VERIFY, TAG).returncode, 0)

    def test_dispatch_targets_release_workflow_in_exact_repository_and_tag(self):
        result = self.run_script(DISPATCH, TAG)
        self.assertEqual(result.returncode, 0, result.stderr)
        self.assertEqual(self.calls.read_text().splitlines(), ["workflow", "run", "release-on-tag.yml",
                         "--repo", "ai-knowledge-hub/ai-skills-guide", "--ref", TAG])

    def test_rejected_source_never_dispatches_and_dispatch_failure_is_not_success(self):
        (self.repo / "payload").write_text("dirty source\n")
        self.assertNotEqual(self.run_script(DISPATCH, TAG).returncode, 0)
        self.assertFalse(self.calls.exists())
        self.git("checkout", "--", "payload")
        self.assertNotEqual(self.run_script(DISPATCH, TAG, RELEASE_TEST_EXIT="1").returncode, 0)
        self.assertTrue(self.calls.exists())


if __name__ == "__main__":
    unittest.main()
