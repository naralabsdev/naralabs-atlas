#!/usr/bin/env bash
set -euo pipefail

check_pkg() {
	local pkg="$1"
	local min="$2"
	local out pct

	if ! out="$(go test "$pkg" -count=1 -covermode=atomic -coverprofile=/tmp/atlas-cover.out 2>&1)"; then
		echo "$out"
		return 1
	fi

	pct="$(echo "$out" | sed -nE 's/.*coverage: ([0-9.]+)% of statements/\1/p' | tail -1)"
	if [[ -z "$pct" ]]; then
		echo "missing coverage for $pkg"
		return 1
	fi

	echo "$pkg: ${pct}% (min ${min}%)"
	if awk -v p="$pct" -v m="$min" 'BEGIN { exit !(p + 0 >= m + 0) }'; then
		return 0
	fi

	echo "FAIL: $pkg coverage ${pct}% below minimum ${min}%"
	return 1
}

failed=0

check_pkg "./config" 60 || failed=1
check_pkg "./lib/rpcchain" 70 || failed=1
check_pkg "./lib/scval" 55 || failed=1
check_pkg "./lib/scval/address" 70 || failed=1
check_pkg "./lib/scval/token" 70 || failed=1
check_pkg "./internal/client/horizon" 80 || failed=1
check_pkg "./internal/client/stellar" 55 || failed=1
check_pkg "./internal/module/ingest/service" 70 || failed=1

if [[ "$failed" -ne 0 ]]; then
	exit 1
fi

echo "All package coverage thresholds satisfied."
