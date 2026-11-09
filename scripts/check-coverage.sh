#!/bin/sh
set -eu
result=$(go test -cover ./internal/domain ./internal/application)
printf '%s\n' "$result"
printf '%s\n' "$result" | awk '/coverage:/ { value=$5; sub(/%$/, "", value); if (value + 0 < 80) failed=1; seen++ } END { if (seen != 2 || failed) exit 1 }'
