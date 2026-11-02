"""Release authority comes from an exact tag on main, never branch names alone."""

import contextlib
import importlib.util
import io
import json
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch


ROOT = Path(__file__).resolve().parents[2]
VERIFY = ROOT / "scripts/verify-cli-release-source.sh"
DISPATCH = ROOT / "scripts/dispatch-cli-release.sh"
TAG = "v0.4.0-alpha.1"
spec = importlib.util.spec_from_file_location("release_assets", ROOT / "scripts/check-cli-release-assets.py")
assets = importlib.util.module_from_spec(spec)
spec.loader.exec_module(assets)


class CLIReleaseAssetTests(unittest.TestCase):
    def query(self, pages):
        response = subprocess.CompletedProcess([], 0, json.dumps(pages), "")
        with patch.dict(os.environ, {"GH_TOKEN": "fixture-not-a-credential"}), \
                patch.object(assets.subprocess, "run", return_value=response) as run:
            assets.check_release_assets("ai-knowledge-hub/ai-skills-guide", TAG)
        return run

    def test_new_tag_and_empty_release_allow_publication(self):
        for pages in [[[]], [[{"tag_name": TAG, "assets": []}]],
                      [[{"tag_name": "v0.3.0", "assets": [{"name": "old-binary"}]}]]]:
            with self.subTest(pages=pages):
                run = self.query(pages)
                args = run.call_args.args[0]
                self.assertIn("--paginate", args)
                self.assertIn("--slurp", args)
                self.assertIn("repos/ai-knowledge-hub/ai-skills-guide/releases?per_page=100", args)
                self.assertTrue(run.call_args.kwargs["check"])

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
        self.query([[{"tag_name": TAG, "assets": []}]])
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
