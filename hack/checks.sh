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

echo "Running validation scripts..."
scripts=(
    "hack/boilerplate.sh"
    "hack/gofmt.sh"
    "hack/linter.sh"
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
    if [[ $result -eq 0 ]]; then
        echo -e "${GREEN}PASSED${RESET} ${s}"
    else
        echo -e "${RED}FAILED${RESET} ${s}"
        fail=1
    fi
done
exit $fail