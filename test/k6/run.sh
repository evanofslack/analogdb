#!/usr/bin/env bash

# Runs a k6 script with a unique testid tag, so each run can be picked out in
# the Grafana dashboard. Results go to Prometheus when
# K6_PROMETHEUS_RW_SERVER_URL is set, and the summary to results/.

set -euo pipefail

if [ $# -lt 1 ]; then
    echo "Usage: ./run.sh <script> [k6 args]"
    exit 1
fi

if ! command -v k6 >/dev/null; then
    echo "k6 not found, install it with: brew install k6"
    exit 1
fi

script=$1
shift
testid="$(basename -s .js "$script")-$(date +%s)"
args=(--tag "testid=$testid" --summary-export "results/$testid.json")

if [ -n "${K6_PROMETHEUS_RW_SERVER_URL:-}" ]; then
    # k6 v2 moved native histograms behind a feature flag
    if [[ "$(k6 version)" =~ k6\ v1\. ]]; then
        export K6_PROMETHEUS_RW_TREND_AS_NATIVE_HISTOGRAM=true
    else
        export K6_FEATURES="${K6_FEATURES:+$K6_FEATURES,}native-histograms"
    fi
    args+=(-o experimental-prometheus-rw)
else
    echo "K6_PROMETHEUS_RW_SERVER_URL is unset, results stay local"
fi

mkdir -p results
k6 version
echo "testid=$testid"
k6 run "${args[@]}" "$@" "$script"
