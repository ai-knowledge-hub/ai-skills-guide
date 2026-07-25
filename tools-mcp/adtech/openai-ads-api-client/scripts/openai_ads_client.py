#!/usr/bin/env python3
"""Read-only OpenAI Advertiser API client with deterministic mock fixtures."""

import argparse
import json
import os
import sys
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

DEFAULT_BASE_URL = "https://api.ads.openai.com/v1"
ROOT = Path(__file__).resolve().parents[1]


class AdsClientError(RuntimeError):
    pass


class AdsClient:
    def __init__(self, mode="mock", api_key=None, base_url=DEFAULT_BASE_URL, fixture_dir=None):
        if mode not in {"mock", "live"}:
            raise ValueError("mode must be mock or live")
        self.mode = mode
        self.api_key = api_key
        self.base_url = base_url.rstrip("/")
        self.fixture_dir = Path(fixture_dir or ROOT / "examples")
        if self.mode == "live" and not self.api_key:
            raise AdsClientError("OPENAI_ADS_API_KEY is required for live mode")

    def get_account(self):
        return self._get("/ad_account", fixture="account.json")

    def list_campaigns(self, limit=20, order="desc"):
        return self._get(
            "/campaigns",
            params={"limit": limit, "order": order},
            fixture="campaigns.json",
        )

    def list_ads(self, ad_group_id, limit=20, order="desc"):
        if not ad_group_id:
            raise AdsClientError("ad_group_id is required")
        return self._get(
            "/ads",
            params={"ad_group_id": ad_group_id, "limit": limit, "order": order},
            fixture="ads.json",
        )

    def get_insights(self, scope, entity_id=None, time_granularity="daily", limit=20):
        if scope == "ad_account":
            path = "/ad_account/insights"
        elif scope in {"campaign", "ad_group", "ad"}:
            if not entity_id:
                raise AdsClientError(f"entity_id is required for {scope} insights")
            plural = {"campaign": "campaigns", "ad_group": "ad_groups", "ad": "ads"}[scope]
            path = f"/{plural}/{urllib.parse.quote(entity_id, safe='')}/insights"
        else:
            raise AdsClientError("scope must be ad_account, campaign, ad_group, or ad")
        return self._get(
            path,
            params={"time_granularity": time_granularity, "limit": limit},
            fixture="insights.json",
        )

    def _get(self, path, params=None, fixture=None):
        if self.mode == "mock":
            if not fixture:
                raise AdsClientError("mock operation has no fixture")
            return json.loads((self.fixture_dir / fixture).read_text())

        query = urllib.parse.urlencode(params or {})
        url = f"{self.base_url}{path}" + (f"?{query}" if query else "")
        request = urllib.request.Request(
            url,
            method="GET",
            headers={"Authorization": f"Bearer {self.api_key}", "Accept": "application/json"},
        )
        try:
            with urllib.request.urlopen(request, timeout=30) as response:
                return json.loads(response.read().decode("utf-8"))
        except urllib.error.HTTPError as exc:
            detail = exc.read().decode("utf-8", errors="replace")
            raise AdsClientError(f"OpenAI Ads API returned HTTP {exc.code}: {detail}") from exc
        except urllib.error.URLError as exc:
            raise AdsClientError(f"OpenAI Ads API request failed: {exc.reason}") from exc


def build_parser():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mode", choices=["mock", "live"], default="mock")
    parser.add_argument("--api-key-env", default="OPENAI_ADS_API_KEY")
    subparsers = parser.add_subparsers(dest="command", required=True)

    subparsers.add_parser("account")

    campaigns = subparsers.add_parser("campaigns")
    campaigns.add_argument("--limit", type=int, default=20)
    campaigns.add_argument("--order", choices=["asc", "desc"], default="desc")

    ads = subparsers.add_parser("ads")
    ads.add_argument("--ad-group-id", required=True)
    ads.add_argument("--limit", type=int, default=20)
    ads.add_argument("--order", choices=["asc", "desc"], default="desc")

    insights = subparsers.add_parser("insights")
    insights.add_argument("--scope", choices=["ad_account", "campaign", "ad_group", "ad"], required=True)
    insights.add_argument("--entity-id")
    insights.add_argument("--time-granularity", choices=["hourly", "daily", "monthly", "none"], default="daily")
    insights.add_argument("--limit", type=int, default=20)
    return parser


def main(argv=None):
    args = build_parser().parse_args(argv)
    try:
        client = AdsClient(
            mode=args.mode,
            api_key=os.environ.get(args.api_key_env),
        )
        if args.command == "account":
            result = client.get_account()
        elif args.command == "campaigns":
            result = client.list_campaigns(args.limit, args.order)
        elif args.command == "ads":
            result = client.list_ads(args.ad_group_id, args.limit, args.order)
        else:
            result = client.get_insights(args.scope, args.entity_id, args.time_granularity, args.limit)
        print(json.dumps(result, indent=2, sort_keys=True))
    except (AdsClientError, ValueError, OSError, json.JSONDecodeError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
