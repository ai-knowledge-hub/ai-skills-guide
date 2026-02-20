#!/usr/bin/env python3
import json, sys

def safe_div(a, b):
    return (a / b) if b else 0.0

def enrich(row):
    spend = float(row.get("spend", 0))
    clicks = float(row.get("clicks", 0))
    impressions = float(row.get("impressions", 0))
    conversions = float(row.get("conversions", 0))
    revenue = float(row.get("revenue", 0))
    row["ctr"] = safe_div(clicks, impressions)
    row["cpc"] = safe_div(spend, clicks)
    row["cvr"] = safe_div(conversions, clicks)
    row["cpa"] = safe_div(spend, conversions)
    row["roas"] = safe_div(revenue, spend)
    return row

if __name__ == "__main__":
    payload = json.load(sys.stdin)
    campaigns = [enrich(c) for c in payload.get("campaigns", [])]
    print(json.dumps({"campaigns": campaigns}, indent=2))
