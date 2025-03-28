#!/usr/bin/env bash
set -e

usage() {
  echo "Usage: "
  echo "  $0 <input-json>"
  echo
  echo "    <input-json> inputs in JSON format"
  echo
  echo "Examples: "
  echo "  $0 '{\"tag\": \"v1.2.3\"}'"
  echo "  $0 '{\"issue_number\": 123}'"
  echo
  exit 1;
}

INPUT_JSON="${1}"
if [[ -z "${INPUT_JSON}" ]]; then
  echo "Must specify input json"
  echo
  usage
  exit 1
fi

tmpEventFile=$(mktemp)
cat <<EOF > "$tmpEventFile"
{
  "action": "workflow_dispatch",
  "inputs": ${INPUT_JSON}
}
EOF

echo "$tmpEventFile"