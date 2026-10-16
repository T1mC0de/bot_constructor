#!/usr/bin/env bash
# Download all orders as JSON.
#
#   core/scripts/download_orders.sh [output.json]   # default: orders.json
#
# Environment: API_URL (default http://localhost:8080).
set -euo pipefail

out="${1:-orders.json}"
curl --silent --show-error --fail-with-body \
  -o "${out}" "${API_URL:-http://localhost:8080}/api/v1/orders"
echo "saved to ${out}"
