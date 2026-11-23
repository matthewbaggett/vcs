#!/usr/bin/env bash
# Coverage floor of the quality gate. Total statement coverage must reach the floor.
#
# Usage:
#   .claude/check-coverage.sh              run the tests with a profile, then check the total
#   .claude/check-coverage.sh --from FILE  check the last line of a `go tool cover -func` report
#
# Exit 0 when the total meets the floor. Exit 1 when it falls below. Exit 2 on a bad input.
# COVERAGE_FLOOR sets the floor. The default is 75.
set -euo pipefail

floor="${COVERAGE_FLOOR:-75}"

if [ "${1:-}" = "--from" ]; then
  report="${2:?usage: check-coverage.sh --from FILE}"
else
  profile="$(mktemp)"
  report="$(mktemp)"
  trap 'rm -f "$profile" "$report"' EXIT
  go test -coverprofile="$profile" ./... >/dev/null
  go tool cover -func="$profile" >"$report"
fi

total="$(awk '$1 == "total:" { sub(/%/, "", $3); print $3 }' "$report")"
if [ -z "$total" ]; then
  echo "check-coverage: no total line in the report" >&2
  exit 2
fi

if awk -v t="$total" -v f="$floor" 'BEGIN { exit !(t + 0 >= f + 0) }'; then
  echo "check-coverage: ${total}% meets the ${floor}% floor"
else
  echo "check-coverage: ${total}% is below the ${floor}% floor" >&2
  exit 1
fi
