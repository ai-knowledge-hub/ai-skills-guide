#!/usr/bin/env python3
"""Bounded, read-only OpenAI Advertiser API client with deterministic mocks."""

import argparse
import json
import os
import re
import sys
import urllib.error
import urllib.parse
import urllib.request
from pathlib import Path

DEFAULT_BASE_URL = "https://api.ads.openai.com/v1"
MAX_RESPONSE_BYTES = 2 * 1024 * 1024
ROOT = Path(__file__).resolve().parents[1]
SAFE_ID = re.compile(r"^[A-Za-z0-9_-]{1,128}$")
SAFE_HEADER_CREDENTIAL = re.compile(r"^[\x21-\x7e]{1,4096}$")


class AdsClientError(RuntimeError):
    """A public, credential-safe client failure."""


class _RejectRedirects(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        raise AdsClientError("OpenAI Ads API redirects are not allowed")


def _validate_base_url(base_url, allow_loopback=False):
    value = base_url.rstrip("/")
    parsed = urllib.parse.urlparse(value)
    if value == DEFAULT_BASE_URL:
        return value
    if allow_loopback and parsed.scheme == "http" and parsed.hostname in {"127.0.0.1", "::1", "localhost"}:
        return value
    raise AdsClientError("OpenAI Ads API endpoint is not trusted")


def _bounded_json(response):
    if response.headers.get_content_type() != "application/json":
        raise AdsClientError("OpenAI Ads API returned an unsupported content type")
    raw = response.read(MAX_RESPONSE_BYTES + 1)
    if len(raw) > MAX_RESPONSE_BYTES:
        raise AdsClientError("OpenAI Ads API response exceeds the safe size limit")
    try:
        value = json.loads(raw.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as exc:
        raise AdsClientError("OpenAI Ads API returned invalid JSON") from exc
    if not isinstance(value, dict):
        raise AdsClientError("OpenAI Ads API returned an invalid response shape")
    return value


def _validated_api_key(value):
    if not isinstance(value, str) or not SAFE_HEADER_CREDENTIAL.fullmatch(value):
        raise AdsClientError("OPENAI_ADS_API_KEY is missing or malformed")
    return value


class AdsClient:
    def __init__(self, mode="mock", api_key=None, base_url=DEFAULT_BASE_URL,
                 fixture_dir=None, timeout=30, allow_loopback=False, opener=None):
        if mode not in {"mock", "live"}:
            raise ValueError("mode must be mock or live")
        if not 1 <= timeout <= 120:
            raise ValueError("timeout must be between 1 and 120 seconds")
        self.mode = mode
        self.api_key = _validated_api_key(api_key) if mode == "live" else None
        self.base_url = _validate_base_url(base_url, allow_loopback)
        self.fixture_dir = Path(fixture_dir or ROOT / "examples")
        self.timeout = timeout
        self.opener = opener or urllib.request.build_opener(_RejectRedirects())

    def get_account(self):
        return self._get("/ad_account", fixture="account.json")

    def list_campaigns(self, limit=20, order="desc"):
        return self._get("/campaigns", params={"limit": self._limit(limit), "order": order},
                         fixture="campaigns.json")

    def list_ads(self, ad_group_id, limit=20, order="desc"):
        entity_id = self._entity_id(ad_group_id, "ad_group_id")
        return self._get("/ads", params={"ad_group_id": entity_id, "limit": self._limit(limit), "order": order},
                         fixture="ads.json")

    def get_insights(self, scope, entity_id=None, time_granularity="daily", limit=20):
        if scope == "ad_account":
            path = "/ad_account/insights"
        elif scope in {"campaign", "ad_group", "ad"}:
            entity_id = self._entity_id(entity_id, "entity_id")
            plural = {"campaign": "campaigns", "ad_group": "ad_groups", "ad": "ads"}[scope]
            path = f"/{plural}/{entity_id}/insights"
        else:
            raise AdsClientError("scope must be ad_account, campaign, ad_group, or ad")
        return self._get(path, params={"time_granularity": time_granularity, "limit": self._limit(limit)},
                         fixture="insights.json")

    @staticmethod
    def _limit(value):
        if isinstance(value, bool) or not isinstance(value, int) or not 1 <= value <= 100:
            raise AdsClientError("limit must be between 1 and 100")
        return value

    @staticmethod
    def _entity_id(value, label):
        if not isinstance(value, str) or not SAFE_ID.fullmatch(value):
            raise AdsClientError(f"{label} must be a safe provider identifier")
        return value

    def _get(self, path, params=None, fixture=None):
        if self.mode == "mock":
            if not fixture:
                raise AdsClientError("mock operation has no fixture")
            try:
                value = json.loads((self.fixture_dir / fixture).read_text())
            except (OSError, json.JSONDecodeError) as exc:
                raise AdsClientError("mock fixture is unavailable or invalid") from exc
            if not isinstance(value, dict):
                raise AdsClientError("mock fixture has an invalid response shape")
            return value

        query = urllib.parse.urlencode(params or {})
        url = f"{self.base_url}{path}" + (f"?{query}" if query else "")
        request = urllib.request.Request(
            url, method="GET",
            headers={"Authorization": f"Bearer {self.api_key}", "Accept": "application/json"},
        )
        try:
            with self.opener.open(request, timeout=self.timeout) as response:
                return _bounded_json(response)
        except AdsClientError:
            raise
        except urllib.error.HTTPError as exc:
            raise AdsClientError(f"OpenAI Ads API returned HTTP {exc.code}") from exc
        except (urllib.error.URLError, TimeoutError, OSError) as exc:
            raise AdsClientError("OpenAI Ads API request failed") from exc


def build_parser():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--mode", choices=["mock", "live"], default="mock")
    parser.add_argument("--api-key-env", default="OPENAI_ADS_API_KEY")
    parser.add_argument("--timeout", type=int, default=30)
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
        client = AdsClient(mode=args.mode, api_key=os.environ.get(args.api_key_env), timeout=args.timeout)
        if args.command == "account":
            result = client.get_account()
        elif args.command == "campaigns":
            result = client.list_campaigns(args.limit, args.order)
        elif args.command == "ads":
            result = client.list_ads(args.ad_group_id, args.limit, args.order)
        else:
            result = client.get_insights(args.scope, args.entity_id, args.time_granularity, args.limit)
        print(json.dumps(result, indent=2, sort_keys=True))
    except (AdsClientError, ValueError) as exc:
        print(f"error: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
