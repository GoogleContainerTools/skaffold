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
VERSION=1.24.0

function install_linter() {
  echo "Installing GolangCI-Lint"
	${DIR}/install_golint.sh -b $GOPATH/bin v$VERSION
}

if ! [ -x "$(command -v golangci-lint)" ] ; then
  install_linter
elif [[ $(golangci-lint --version | grep -c " $VERSION ") -eq 0 ]]
then
  echo "required golangci-lint: v$VERSION"
  echo "current version: $(golangci-lint --version)"
  echo "reinstalling..."
  rm $(which golangci-lint)
  install_linter
fi

FLAGS=""
if [[ "${TRAVIS}" == "true" ]]; then
    # Use less memory on Travis
    # See https://github.com/golangci/golangci-lint#memory-usage-of-golangci-lint
    export GOGC=5
    FLAGS="-j1 -v --print-resources-usage"
fi

$GOPATH/bin/golangci-lint run ${FLAGS} -c ${DIR}/golangci.yml \
    | awk '/out of memory/ || /Timeout exceeded/ {failed = 1}; {print}; END {exit failed}'
