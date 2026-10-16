#!/usr/bin/env bash
# Upload a survey JSON file to core.
#
#   core/scripts/upload_survey.sh core/examples/survey_v1.json
#
# Environment: API_URL (default http://localhost:8080).
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <survey.json>" >&2
  exit 2
fi

curl --silent --show-error \
  --write-out '\nHTTP %{http_code}\n' \
  -X PUT "${API_URL:-http://localhost:8080}/api/v1/survey" \
  -H 'Content-Type: application/json' \
  --data-binary "@$1"
