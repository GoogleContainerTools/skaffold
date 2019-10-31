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
BIN=${DIR}/bin
LICENSES=${BIN}/licenses
STATIK=${BIN}/statik

mkdir -p ${BIN}

if ! [[ -f ${LICENSES} ]]; then
  pushd ${DIR}/tools
  echo >&2 'Installing licenses tool'
  GOBIN=${BIN} GO111MODULE=on go get github.com/google/trillian/scripts/licenses
  popd
fi

TMP_DIR=$(mktemp -d)
${LICENSES} save "github.com/GoogleContainerTools/skaffold/cmd/skaffold" --save_path="${TMP_DIR}/skaffold-credits"

if ! [[ -f ${STATIK} ]]; then
  pushd ${DIR}/tools
  echo >&2 'Installing statik tool'
  GOBIN=${BIN} GO111MODULE=on go get github.com/rakyll/statik
  popd
fi

${STATIK} -f -src=${TMP_DIR}/skaffold-credits/ -m -dest cmd/skaffold/app/cmd/credits
