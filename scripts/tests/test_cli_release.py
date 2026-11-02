"""Release authority comes from an exact tag on main, never branch names alone."""

import os
from pathlib import Path
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
VERIFY = ROOT / "scripts/verify-cli-release-source.sh"
DISPATCH = ROOT / "scripts/dispatch-cli-release.sh"
TAG = "v0.4.0-alpha.1"


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
