#!/usr/bin/env bash
# Seeds the CV on a deployed API through the admin endpoint (PUT /api/v1/admin/cv),
# so no database tunnel is needed. Replaces whatever CV is stored.
#
#   scripts/seed-remote.sh https://maulanaiskak-api.onrender.com you@example.com [seed/cv.json]
set -euo pipefail

API="${1:?usage: seed-remote.sh <api-base-url> <admin-email> [cv.json]}"
API="${API%/}"
export EMAIL="${2:?admin email required}"
FILE="${3:-seed/cv.json}"

read -rsp "Admin password: " PW && echo
export PW

# Password and token go through env/stdin, never argv (visible in `ps`).
token=$(jq -n '{email: env.EMAIL, password: env.PW}' |
  curl -fsS -H 'Content-Type: application/json' --data @- "$API/api/v1/auth/login" |
  jq -r '.data.token')
unset PW

curl -fsS -X PUT -H 'Content-Type: application/json' \
  -H @<(printf 'Authorization: Bearer %s\n' "$token") \
  --data @"$FILE" "$API/api/v1/admin/cv" |
  jq '{response_code, experiences: (.data.experiences | length), projects: (.data.projects | length), skills: (.data.skills | length)}'
