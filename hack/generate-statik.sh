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

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

BIN=${DIR}/bin
STATIK=${BIN}/statik
LICENSES=${BIN}/go-licenses

TMP_DIR=$(mktemp -d ${TMPDIR:-/tmp}/generate-statik.XXXXXX)
trap "rm -rf $TMP_DIR" EXIT

if [ -x "$(command -v go-licenses)" ]; then
    # use go-licenses binary if it's installed on user's path
    LICENSES=go-licenses
elif ! [ -x "$(command -v ${LICENSES})" ]; then
    # See https://github.com/golang/go/issues/30515
    # Also can't be easily installed from a vendor folder because it relies on non-go files
    # from a dependency.
    echo "Installing go-licenses"
    pushd $(mktemp -d ${TMPDIR:-/tmp}/generate-statik.XXXXXX)
    go mod init tmp; GOBIN=${BIN} go get github.com/google/go-licenses
    popd
fi

echo "Collecting licenses"
cd ${DIR}/..
${LICENSES} save github.com/GoogleContainerTools/skaffold/cmd/skaffold --save_path="${TMP_DIR}/skaffold-credits"
chmod -R u+w "${TMP_DIR}/skaffold-credits"

echo "Collecting schemas"
cp -R docs/content/en/schemas "${TMP_DIR}/schemas"

if ! [[ -f ${STATIK} ]]; then
    echo 'Installing statik tool'
    pushd ${DIR}/tools
    GOBIN=${BIN} GO111MODULE=on go install -mod=vendor -tags tools github.com/rakyll/statik
    popd
fi

${STATIK} -f -src=${TMP_DIR} -m -dest cmd/skaffold/app/cmd
