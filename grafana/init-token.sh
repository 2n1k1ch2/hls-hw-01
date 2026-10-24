#!/bin/sh
set -eu

BASE="http://grafana:3000"
NAME="shop-dashboard-generator"

apk add --no-cache curl jq

RESPONSE=$(curl -fsS -u "$GRAFANA_ADMIN_USER:$GRAFANA_ADMIN_PASSWORD" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$NAME\",\"role\":\"Editor\"}" \
  "$BASE/api/serviceaccounts" || true)

if [ -z "$RESPONSE" ]; then
  RESPONSE=$(curl -fsS -u "$GRAFANA_ADMIN_USER:$GRAFANA_ADMIN_PASSWORD" \
    "$BASE/api/serviceaccounts/search?query=$NAME")
  ID=$(echo "$RESPONSE" | jq -r \
    --arg name "$NAME" '.serviceAccounts[] | select(.name == $name) | .id' \
    | head -n 1)
else
  ID=$(echo "$RESPONSE" | jq -r '.id')
fi

test -n "$ID" && test "$ID" != "null"

TOKEN=$(curl -fsS -u "$GRAFANA_ADMIN_USER:$GRAFANA_ADMIN_PASSWORD" \
  -H "Content-Type: application/json" \
  -d '{"name":"dashboard-generator"}' \
  "$BASE/api/serviceaccounts/$ID/tokens" | jq -r '.key')

test -n "$TOKEN" && test "$TOKEN" != "null"
printf '%s' "$TOKEN" > /token/token
chmod 600 /token/token