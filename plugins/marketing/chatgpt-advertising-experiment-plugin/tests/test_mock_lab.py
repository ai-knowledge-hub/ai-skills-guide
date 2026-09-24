import json
import subprocess
import sys
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]


class MockLabTests(unittest.TestCase):
    def test_smoke_check_runs_without_writing_output(self):
        output = ROOT / "output" / "mock-evidence-bundle.json"
        output.unlink(missing_ok=True)
        result = subprocess.run(
            [sys.executable, str(ROOT / "scripts" / "run_mock_lab.py"), "--check"],
            check=True,
            capture_output=True,
            text=True,
        )
        self.assertEqual(json.loads(result.stdout), {"network_calls": 0, "valid": True})
        self.assertFalse(output.exists())

    def test_mock_lab_builds_offline_evidence_bundle(self):
        result = subprocess.run(
            [sys.executable, str(ROOT / "scripts" / "run_mock_lab.py")],
            check=True,
            capture_output=True,
            text=True,
        )
        output = Path(result.stdout.strip())
        payload = json.loads(output.read_text())
        self.assertEqual(payload["network_calls"], 0)
        self.assertTrue(payload["conversion_validation"]["valid"])
        self.assertEqual(payload["reconciliation"]["summary"]["verified_outcome_count"], 1)
        output.unlink()
        output.parent.rmdir()


if __name__ == "__main__":
    unittest.main()
