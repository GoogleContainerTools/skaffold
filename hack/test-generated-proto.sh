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

docker build -t gen-proto -f hack/proto/Dockerfile --target compare proto
docker run --rm gen-proto
if [ $? -ne 0 ]; then
   printf "\nGenerated proto files aren't updated. Please run ./hack/generate-proto.sh\n"
   exit 1
fi

printf "\nGenerated proto files are updated!\n"
