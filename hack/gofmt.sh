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

files=$(find . -name "*.go" | grep -v vendor/ | xargs gofmt -l -s)
if [[ $files ]]; then
    echo "Gofmt errors in files:"
    echo "$files"
    diff=$(find . -name "*.go" | grep -v vendor/ | xargs gofmt -d -s)
    echo "$diff"
    exit 1
fi
