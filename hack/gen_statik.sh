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

export GOFLAGS="-mod=vendor"
DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
BIN=${DIR}/bin
STATIK=${BIN}/statik

mkdir -p ${BIN}

TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Copy licenses
pushd vendor
LICENSES=$(find . \( -type f -name 'LICENSE*' -or -name 'COPYING*' -or -name 'NOTICE*' \))
for LICENSE in $LICENSES; do
    mkdir -p "$(dirname "${TMP_DIR}/skaffold-credits/$LICENSE")"
    cp $LICENSE ${TMP_DIR}/skaffold-credits/$LICENSE
done
popd

# Copy schemas
cp -R docs/content/en/schemas "${TMP_DIR}/schemas"

if ! [[ -f ${STATIK} ]]; then
  pushd ${DIR}/tools
  echo >&2 'Installing statik tool'
  GOBIN=${BIN} GO111MODULE=on go install -tags tools github.com/rakyll/statik
  popd
fi

${STATIK} -f -src=${TMP_DIR} -m -dest cmd/skaffold/app/cmd
