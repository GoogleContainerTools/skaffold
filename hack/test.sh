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
RESET='\033[0m'

echo "Running go tests..."
go test -count=1 -race -cover -short -timeout=90s -coverprofile=out/coverage.txt -coverpkg="./pkg/...,./cmd/..." ./... | awk -v FAIL="${RED}FAIL${RESET}" '! /no test files/ { gsub("FAIL", FAIL, $0); print $0 }'

exit ${PIPESTATUS[0]}
