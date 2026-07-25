import importlib.util
import json
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SPEC = importlib.util.spec_from_file_location(
    "reconcile_events", ROOT / "scripts" / "reconcile_events.py"
)
MODULE = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(MODULE)


class ReconcileEventsTests(unittest.TestCase):
    def test_bundled_fixture(self):
        payload = json.loads((ROOT / "examples" / "input.json").read_text())
        result = MODULE.reconcile(payload)

        self.assertEqual(result["summary"]["raw_event_count"], 4)
        self.assertEqual(result["summary"]["canonical_event_count"], 3)
        self.assertEqual(result["summary"]["duplicate_event_count"], 1)
        self.assertEqual(result["summary"]["verified_outcome_count"], 1)
        self.assertEqual(result["summary"]["reversed_outcome_count"], 1)
        self.assertTrue(any(item["code"] == "missing_event_id" for item in result["warnings"]))

    def test_conflicting_duplicate_values_are_reported(self):
        payload = {
            "browser_events": [
                {"event_name": "purchase", "event_id": "evt-1", "value": 20, "currency": "GBP"}
            ],
            "server_events": [
                {"event_name": "purchase", "event_id": "evt-1", "value": 25, "currency": "GBP"}
            ],
        }
        result = MODULE.reconcile(payload)

        self.assertEqual(result["summary"]["canonical_event_count"], 1)
        self.assertEqual(result["canonical_events"][0]["conflicts"][0]["field"], "value")
        self.assertTrue(any(item["code"] == "event_conflict" for item in result["warnings"]))

    def test_server_event_enriches_missing_browser_fields(self):
        payload = {
            "browser_events": [
                {"event_name": "purchase", "event_id": "evt-2"}
            ],
            "server_events": [
                {
                    "event_name": "purchase",
                    "event_id": "evt-2",
                    "order_id": "ord-2",
                    "value": 42,
                    "currency": "GBP",
                }
            ],
            "outcomes": [{"order_id": "ord-2", "status": "settled"}],
        }
        result = MODULE.reconcile(payload)
        event = result["canonical_events"][0]

        self.assertEqual(event["order_id"], "ord-2")
        self.assertEqual(event["value"], 42)
        self.assertEqual(event["currency"], "GBP")
        self.assertEqual(event["conflicts"], [])
        self.assertEqual(result["summary"]["verified_outcome_count"], 1)
        self.assertFalse(any(item["code"] == "event_conflict" for item in result["warnings"]))


if __name__ == "__main__":
    unittest.main()
