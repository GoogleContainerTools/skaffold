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

DIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
EXAMPLES_DIR=${DIR}/../examples
INTEGRATION_EXAMPLES_DIR=${DIR}/../integration/examples

install_release_notes_helper() {
  release_notes_workdir="$(mktemp -d)"
  trap 'rm -rf -- ${release_notes_workdir}' RETURN

  # See https://stackoverflow.com/questions/56842385/using-go-get-to-download-binaries-without-adding-them-to-go-mod for this workaround
  cd "${release_notes_workdir}"
  go mod init release-notes
  GOBIN="$DIR" go get github.com/corneliusweig/release-notes
  cd -
}

if ! [[ -x "${DIR}/release-notes" ]]; then
  echo >&2 'Installing release-notes'
  install_release_notes_helper
fi

# you can pass your github token with --token here if you run out of requests
"${DIR}/release-notes" GoogleContainerTools skaffold

# sync files from integration examples to examples/
rm -rf ${EXAMPLES_DIR} && rm -rf ${INTEGRATION_EXAMPLES_DIR}/bazel/bazel-* && cp -r ${INTEGRATION_EXAMPLES_DIR} ${EXAMPLES_DIR} && rm -rf ${EXAMPLES_DIR}/test-*

go run hack/versions/cmd/mark_latest_released/main.go

echo
echo "Huge thanks goes out to all of our contributors for this release:"
echo
git log "$(git describe  --abbrev=0)".. --format="%aN" --reverse | sort | uniq | awk '{printf "- %s\n", $0 }'
echo
