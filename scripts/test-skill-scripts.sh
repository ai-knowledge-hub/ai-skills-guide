#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "[check] meta/google metrics script"
meta_out="$(
  cat <<'JSON' | python3 "$ROOT/skills/marketing/meta-google-weekly-performance-review/scripts/compute_metrics.py"
{"campaigns":[{"spend":100,"clicks":50,"impressions":1000,"conversions":5,"revenue":250}]}
JSON
)"
echo "$meta_out" | grep -q '"roas": 2.5'

echo "[check] policy + brand UTM lint script"
utm_out="$(
  cat <<'JSON' | python3 "$ROOT/skills/adtech/policy-brand-compliance-checker/scripts/utm_lint.py"
{"urls":["https://example.com/?utm_source=google&utm_medium=cpc&utm_campaign=brand-test","https://example.com/?utm_source=google"]}
JSON
)"
echo "$utm_out" | grep -q '"valid": true'
echo "$utm_out" | grep -q '"valid": false'

echo "[check] creative length validator"
len_out="$(
  cat <<'JSON' | python3 "$ROOT/skills/marketing/creative-workshop-pmax-reels/scripts/validate_lengths.py"
{"pmax_headline":"Great deal","pmax_description":"Short and clear description","reels_primary":"Punchy reels intro"}
JSON
)"
echo "$len_out" | grep -q '"valid": true'

echo "[check] seo winners extractor"
seo_out="$(
  cat <<'JSON' | python3 "$ROOT/skills/marketing/seo-paid-search-synergy/scripts/extract_seo_winners.py"
{"queries":[{"query":"best running shoes","ctr":0.12,"clicks":180},{"query":"brand x","ctr":0.03,"clicks":300}]}
JSON
)"
echo "$seo_out" | grep -q 'best running shoes'

echo "[check] query safety script"
echo 'select * from t' | python3 "$ROOT/skills/adtech/analyst-copilot-bigquery-redshift/scripts/query_safety_check.py" >/dev/null
if echo 'drop table t' | python3 "$ROOT/skills/adtech/analyst-copilot-bigquery-redshift/scripts/query_safety_check.py" >/dev/null 2>&1; then
  echo "[ERROR] query_safety_check.py did not block destructive SQL"
  exit 1
fi

echo "[check] lifecycle sample size script (json contract + numeric output)"
sample_out="$(
  cat <<'JSON' | python3 "$ROOT/skills/marketing/lifecycle-experiment-planner/scripts/sample_size.py"
{"baseline_rate":0.04,"minimum_detectable_effect":0.005,"confidence_level":95,"power":80}
JSON
)"
echo "$sample_out" | grep -q '"recommended_total_sample_size"'

echo "All skill scripts passed deterministic checks."
