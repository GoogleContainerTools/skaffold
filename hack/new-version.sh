#!/usr/bin/env bash

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

go run ./hack/versions/cmd/new/version.go $@

goimports -w ./pkg/skaffold/schema
make generate-schemas
./hack/generate-man.sh
git --no-pager diff --minimal
make quicktest

echo
echo "---------------------------------------"
echo
echo "Files generated for the new version."
echo "All tests should have passed. For the docs change, commit the results and rerun 'make test'."
echo "Please double check manually the generated files as well: the upgrade functionality, and all the examples:"
echo
git status -s
echo
echo "---------------------------------------"
