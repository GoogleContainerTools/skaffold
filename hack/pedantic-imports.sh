#!/usr/bin/env bash

# Copyright 2020 The Skaffold Authors
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

EXIT_CODE=0

for gofile in $(find . -name *.go | grep -v '/vendor/'); do
    awk '{
        if ($0 == "import (") {inImport=1}
        if (inImport && $0 == "") {blankLines++}
        if ($0 == ")") {inImport=0; exit}
    } END {
        if (blankLines > 2) {exit 1}
    }' "${gofile}"
    if [[ $? -ne 0 ]]; then
        echo "${gofile} contains more than 3 groups of imports"
        EXIT_CODE=1
    fi
    
    awk '{
        if ($0 == "import (") {inImport=1}
        if (inImport && $0 == "") {blankLines++}
        if (inImport && $0 != ")") {last=$0}
        if ($0 == ")") {inImport=0; exit}
    } END {
        if (blankLines == 2 && index(last, "github.com/GoogleContainerTools") == 0) {exit 1}
    }' "${gofile}"
    if [[ $? -ne 0 ]]; then
        echo "${gofile} should have skaffold imports last"
        EXIT_CODE=1
    fi
done

exit ${EXIT_CODE}
