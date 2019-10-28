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

export GOFLAGS=""
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
LICENSES=${DIR}/licenses
STATIK=${DIR}/statik


if ! [[ -f ${LICENSES} ]]; then
  echo >&2 'Installing licenses tool'
  GOBIN=${DIR} GO111MODULE=on go get github.com/google/trillian/scripts/licenses@c93851d711b5
fi

TMP_DIR=$(mktemp -d)
${LICENSES} save "github.com/GoogleContainerTools/skaffold/cmd/skaffold" --save_path="${TMP_DIR}/skaffold-credits"

OUT_DIR=./out/third-party-notices
mkdir -p ${OUT_DIR}

tar -cz -C "${TMP_DIR}" -f ${OUT_DIR}/licenses.tgz  .

if ! [[ -f ${STATIK} ]]; then
  echo >&2 'Installing statik tool'
  GOBIN=${DIR} GO111MODULE=on go get github.com/rakyll/statik
fi

${STATIK} -src=${TMP_DIR} -m -dest cmd/skaffold/app/cmd/credits
