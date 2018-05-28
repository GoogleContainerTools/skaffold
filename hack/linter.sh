#!/bin/bash

# Copyright 2018 The Skaffold Authors
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

if ! [ -x "$(command -v golangci-lint)" ]; then
	echo "Installing GolangCI-Lint"
	go get -u github.com/golangci/golangci-lint/cmd/golangci-lint
fi

golangci-lint run \
	--no-config \
	-E goimports \
	-E interfacer \
	-E unconvert \
	-E goconst \
	-E maligned \
	-D errcheck

# From now on, run go lint.
golangci-lint run \
	--disable-all \
	-E golint \
	--new-from-rev bed41e9a77431990cc8504c0955252c851934b89