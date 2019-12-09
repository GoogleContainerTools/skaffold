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

set -euo pipefail

DIR=$(mktemp -d)
trap "rm -rf $DIR" EXIT

cd "${DIR}"
curl -sSLf https://github.com/GoogleCloudPlatform/cloud-code-samples/tarball/master | tar -xz --strip-components=1

for SKAFFOLD_YAML in $(find . -name skaffold.yaml); do
    PROJECT_DIR="$(dirname "${SKAFFOLD_YAML}")"

    pushd "${PROJECT_DIR}"
        skaffold build
    popd
done
