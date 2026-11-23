#!/usr/bin/env bash
# Test of check-coverage.sh. It proves the check fails below the floor and passes at it.
set -uo pipefail

here="$(cd "$(dirname "$0")" && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fail=0

report() { printf 'pkg/a.go:1:\tf\t100.0%%\ntotal:\t(statements)\t%s%%\n' "$1" >"$tmp/r"; }

expect() { # name, want exit, floor, total
  report "$4"
  COVERAGE_FLOOR="$3" "$here/check-coverage.sh" --from "$tmp/r" >/dev/null 2>&1
  got=$?
  if [ "$got" -ne "$2" ]; then
    echo "FAIL $1: want exit $2, got $got"
    fail=1
  else
    echo "ok   $1"
  fi
}

expect "below the floor fails" 1 75 74.9
expect "at the floor passes" 0 75 75.0
expect "above the floor passes" 0 75 90.2
expect "default floor is 75, 74 fails" 1 "" 74.0

printf 'no total here\n' >"$tmp/r"
"$here/check-coverage.sh" --from "$tmp/r" >/dev/null 2>&1
if [ $? -ne 2 ]; then
  echo "FAIL a report with no total must exit 2"
  fail=1
else
  echo "ok   a report with no total exits 2"
fi

exit "$fail"
