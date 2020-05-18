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

RED='\033[0;31m'
GREEN='\033[0;32m'
RESET='\033[0m'

echo "Running validation scripts..."
scripts=(
    "hack/check-schema-changes.sh"
    "hack/check-skaffold-builder.sh"
    "hack/check-samples.sh"
    "hack/check-docs.sh"
    "hack/test-generated-proto.sh"
)
fail=0
for s in "${scripts[@]}"; do
    echo "RUN ${s}"

    start=$(date +%s)
    ./$s
    result=$?
    end=$(date +%s)

    if [[ $result -eq 0 ]]; then
        echo -e "${GREEN}PASSED${RESET} ${s} in $((end-start))s"
    else
        echo -e "${RED}FAILED${RESET} ${s}"
        fail=1
    fi
done
exit $fail
