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

## This tool should only be used when the hack/tools/vendor directory needs update
## Why are we using this instead of go mod vendor? Because https://github.com/golang/go/issues/32502
## What files do we need?
## vendor/github.com/google/licenseclassifier/licenses - it contains no go files, hence it's not part of it
## vendor/github.com/google/trillian/scripts/licenses - it has no go.mod for it

DIR="$(cd $(dirname "${BASH_SOURCE[0]}") && pwd)"
cd ${DIR}
go mod vendor

git clone https://github.com/google/licenseclassifier
cd licenseclassifier
git checkout 842c0d70d702
cd ${DIR}
cp -R licenseclassifier/licenses vendor/github.com/google/licenseclassifier/
rm -rf licenseclassifier

git clone https://github.com/google/trillian
cd trillian
git checkout 9600d042b2e7
mkdir -p vendor/github.com/google/trillian/scripts/
cd ${DIR}
cp -R trillian/scripts/licenses vendor/github.com/google/trillian/scripts/
rm -rf trillian


