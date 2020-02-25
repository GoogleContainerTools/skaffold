#!/bin/bash

# Copyright 2019 The Skaffold Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -e

# This script runs go test with a better output:
# - It prints the failures in RED
# - It recaps the failures at the end
# - It lists the 20 slowest tests

BOLD='\033[1m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
RESET='\033[0m'

LOG=$(mktemp -t tests.json.XXXXXX)
trap "rm -f $LOG" EXIT

if [[ " ${@}" =~ "pkg/skaffold" ]]; then
  echo "go test ./pkg/skaffold/..."
else
  echo "go test $@"
fi

# Keep execution simple for users who do not have jq installed
if [[ ! $(command -v jq >/dev/null) ]]; then
  go test $* | sed ''/FAIL/s//`printf "${RED}FAIL${RESET}"`/''
  exit ${PIPESTATUS[0]}
fi

if [[ " ${@} " =~ "-v" ]]; then
    JQ_FILTER='select(has("Output") and (.Action=="output")) | .Output'
else
    JQ_FILTER='select(has("Output") and (.Action=="output") and (has("Test")|not) and (.Output!="PASS\n") and (.Output!="FAIL\n") and (.Output|startswith("coverage:")|not) and (.Output|contains("[no test files]")|not)) | .Output'
fi

go test -json $* | tee $LOG | jq --unbuffered -j "${JQ_FILTER}" | sed ''/FAIL/s//`printf "${RED}FAIL${RESET}"`/''
RESULT=${PIPESTATUS[0]}

if [ $RESULT != 0 ]; then
  MODULE="$(go list -m)"
  echo -e "\n${RED}=== Failed Tests ===${RESET}"

  FAILED_TESTS=$(cat $LOG | jq -r 'select(.Action=="fail" and has("Test")) | "\(.Package) \(.Test)"')
  while IFS= read -r line; do
    ID=( $line )
    PACKAGE_NAME=${ID[0]}
    TRIMMED_PACKAGE_NAME=${PACKAGE_NAME#"$MODULE"}
    TEST_NAME=${ID[1]}
    echo -e "${BOLD}$TRIMMED_PACKAGE_NAME/$TEST_NAME${RESET}"
    JQ_FILTER="select(.Action==\"output\" and has(\"Test\") and .Package==\"$PACKAGE_NAME\" and .Test==\"$TEST_NAME\" and has(\"Output\") and (.Output|startswith(\"=== RUN\")|not)) | \"\(.Output|rtrimstr(\"\\n\"))\""
    cat $LOG | jq -r "${JQ_FILTER}"
  done <<< "$FAILED_TESTS"
fi

echo -e "\n${YELLOW}=== Slow Tests ===${RESET}"
cat $LOG | jq -rs 'map(select(.Elapsed > 0 and has("Test"))) | sort_by(.Elapsed) | reverse | map("\(.Elapsed)\t\(.Test)")[]' | head -n20

exit $RESULT
