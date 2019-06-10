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

if ! [ -x "$(command -v golangci-lint)" ]; then
	echo "Installing GolangCI-Lint"
	${DIR}/install_golint.sh -b $GOPATH/bin v1.17.1
fi

golangci-lint run \
	--no-config \
	-E goconst \
	-E goimports \
	-E gocritic \
	-E golint \
	-E interfacer \
	-E maligned \
	-E misspell \
	-E stylecheck \
	-E unconvert \
	-E unparam \
	-D errcheck \
  --skip-dirs vendor
