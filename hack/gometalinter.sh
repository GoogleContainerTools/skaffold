#!/bin/bash

# Copyright 2018 Google LLC
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


#!/bin/bash
set -e -o pipefail

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

install_gometalinter() {
	echo "Installing gometalinter.v2"
	go get -u gopkg.in/alecthomas/gometalinter.v2
	gometalinter.v2 --install
}

if ! [ -x "$(command -v gometalinter.v2)" ]; then
  install_gometalinter
fi

gometalinter.v2 \
	${GOMETALINTER_OPTS:--deadine 5m} \
	--config $SCRIPTDIR/gometalinter.json ./...
