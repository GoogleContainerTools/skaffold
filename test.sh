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

RED='\033[0;31m'
GREEN='\033[0;32m'
RESET='\033[0m'

echo "Running go tests..."
go test -count=1 -race -cover -short -timeout=60s -coverprofile=out/coverage.txt -coverpkg="./pkg/...,./cmd/..." ./... | awk -v FAIL="${RED}FAIL${RESET}" '! /no test files/ { gsub("FAIL", FAIL, $0); print $0 }'

GO_TEST_EXIT_CODE=${PIPESTATUS[0]}
if [[ $GO_TEST_EXIT_CODE -ne 0 ]]; then
    exit $GO_TEST_EXIT_CODE
fi

echo "Running validation scripts..."
scripts=(
    "hack/boilerplate.sh"
    "hack/gofmt.sh"
    "hack/linter.sh"
    "hack/dep.sh"
    "hack/check-samples.sh"
    "hack/check-docs.sh"
    "hack/test-generated-proto.sh"
)
fail=0
for s in "${scripts[@]}"; do
    echo "RUN ${s}"
    set +e
    ./$s
    result=$?
    set -e
    if [[ $result  -eq 1 ]]; then
        echo -e "${RED}FAILED${RESET} ${s}"
        fail=1
    else
        echo -e "${GREEN}PASSED${RESET} ${s}"
    fi
done
exit $fail
