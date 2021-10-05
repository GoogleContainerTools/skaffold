#!/bin/bash

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

set -e -o pipefail

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BIN=${DIR}/bin
VERSION=1.37.1

function install_linter() {
  echo "Installing GolangCI-Lint"
  ${DIR}/install-golint.sh -b ${BIN} v$VERSION
}

if ! [ -x "$(command -v ${BIN}/golangci-lint)" ] ; then
  install_linter
elif [[ $(${BIN}/golangci-lint --version | grep -c " $VERSION ") -eq 0 ]]
then
  echo "required golangci-lint: v$VERSION"
  echo "current version: $(golangci-lint --version)"
  echo "reinstalling..."
  rm $(which ${BIN}/golangci-lint)
  install_linter
fi

FLAGS=""
if [[ "${CI}" == "true" ]]; then
    FLAGS="-v --print-resources-usage"
fi

${BIN}/golangci-lint run ${FLAGS} --exclude=SA1019 -c ${DIR}/golangci.yml \
    | awk '/out of memory/ || /Timeout exceeded/ {failed = 1}; {print}; END {exit failed}'


# Install and run custom linter to detect usage for logrus package.
# Currently, we can't run private custom linter in golangcl-lint due to abandoned issue
# https://github.com/golangci/golangci-lint/issues/1276
if ! [ -x "$(command -v ${BIN}/logrus-analyzer)" ] ; then
  echo >&2 'Installing custom logrus linter'
  cd "${DIR}/tools"
  GO111MODULE=on go build -o ${BIN}/logrus-analyzer logrus_analyzer.go
  cd -
fi
# This analyzer doesn't support any flags currently, so we don't include ${FLAGS}
${BIN}/logrus-analyzer github.com/GoogleContainerTools/skaffold{/pkg,/cmd,/diag}...

