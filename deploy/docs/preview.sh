#!/usr/bin/env bash

# Copyright 2018 The Skaffold Authors
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

## This script starts a preview of the Skaffold site assuming it's ran from a
## cloned Skaffold repo with no submodules initialized. The script initializes the git submodules for
## the site's theme in a standard manner, thus this script can be used locally as well as for the PR review flow.
set -x

readonly REPO_DIR=$(pwd)
readonly BASE_URL=${1:-"http://localhost:1313"}

bash ${REPO_DIR}/deploy/docs/build.sh ${BASE_URL}

cd ${REPO_DIR}/docs

hugo serve --bind=0.0.0.0 -D --baseURL=${BASE_URL}
