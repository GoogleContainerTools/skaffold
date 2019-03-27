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

readonly CURRENT_DIR=$(pwd)
readonly DOCS_DIR="${CURRENT_DIR}/docs"

MOUNTS="-v ${CURRENT_DIR}/.git:/app/.git:ro"
MOUNTS="${MOUNTS} -v ${DOCS_DIR}/config.toml:/app/docs/config.toml:ro"

for dir in $(find ${DOCS_DIR} -mindepth 1 -maxdepth 1 -type d | grep -v themes | grep -v public | grep -v resources | grep -v node_modules); do
    MOUNTS="${MOUNTS} -v $dir:/app/docs/$(basename $dir):ro"
done

docker build -t skaffold-docs-previewer --target runtime_deps deploy/webhook
docker run --rm -ti -p 1313:1313 ${MOUNTS} skaffold-docs-previewer $@
