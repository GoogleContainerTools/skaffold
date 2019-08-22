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

if [[ "${TRAVIS}" == "true" ]] && [[ "${TRAVIS_OS_NAME}" != "linux" ]]; then
    printf "On Travis CI, we only test proto generation on Linux\n"
    exit 0
fi

./hack/generate-dash.sh

readonly DASH_CHANGES=`git diff | grep "statik.go" | wc -l`

if [[ ${DASH_CHANGES} -gt 0 ]]; then
  echo "There are changes in skaffold-dash, please run hack/generate-dash.sh and commit the resutls!"
  git diff
  exit 1
fi

exit 0
