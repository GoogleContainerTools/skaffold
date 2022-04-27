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

SECRET=${SECRET:-${DIR}/../secrets}
BIN=${DIR}/bin
LICENSES=${BIN}/go-licenses


if [ -x "$(command -v go-licenses)" ]; then
    # use go-licenses binary if it's installed on user's path
    LICENSES=go-licenses
elif ! [ -x "$(command -v ${LICENSES})" ]; then
    # See https://github.com/golang/go/issues/30515
    # Also can't be easily installed from a vendor folder because it relies on non-go files
    # from a dependency.
   echo "Installing go-licenses"
     pushd $(mktemp -d ${TMPDIR:-/tmp}/generate-embedded.XXXXXX)
     #TODO(marlongamez): unpin this version once we're able to build using go 1.16.x
     go mod init tmp; GOBIN=${BIN} go get github.com/google/go-licenses@9376cf9847a05cae04f4589fe4898b9bce37e684
     popd
fi

echo "Collecting licenses"
cd ${DIR}/..
${LICENSES} save github.com/GoogleContainerTools/skaffold/cmd/skaffold --save_path="fs/assets/credits_generated" --force
chmod -R u+w "fs/assets/credits_generated"

echo "Collecting schemas"
cp -R docs/content/en/schemas "fs/assets/schemas_generated"


if [[ -d ${SECRET} ]]; then
   echo "generating embedded files for secret"
   cp -R ${SECRET} "fs/assets/secrets_generated"
fi

echo "Used for marking generating embedded files task complete, don't modify this." > fs/assets/check.txt
