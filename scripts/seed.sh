#!/bin/sh
set -eu
api_url=${API_URL:-http://localhost:8080}
curl --fail-with-body --silent --show-error "$api_url/api/v1/links" \
  -H 'Content-Type: application/json' \
  -H 'Idempotency-Key: seed-linklet-home-001' \
  -d '{"url":"https://example.com/","code":"welcome1"}'
printf '\n'
