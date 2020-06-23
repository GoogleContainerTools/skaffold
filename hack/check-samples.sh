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

set -e -o pipefail

EXAMPLES=`mktemp ${TMPDIR:-/tmp}/check-samples.XXXXXX`
INTEGRATION_EXAMPLES=`mktemp ${TMPDIR:-/tmp}/check-samples.XXXXXX`

find examples -mindepth 1 -maxdepth 1 -type d -not -empty -exec basename {} \; | sort > $EXAMPLES
find integration/examples -mindepth 1 -maxdepth 1 -type d -not -empty -exec basename {} \; | sort > $INTEGRATION_EXAMPLES

EXAMPLES_MINUS_INTEGRATION_EXAMPLES=`awk 'FNR==NR{ array[$0];next} {if ( $1 in array ) next; print $1}' $INTEGRATION_EXAMPLES $EXAMPLES`

if [ ! -z "$EXAMPLES_MINUS_INTEGRATION_EXAMPLES" ]; then
  echo "Every code sample that is in ./examples should also be in ./integration/examples"
  echo "The following are in examples but not in integration/examples:"
  echo $EXAMPLES_MINUS_INTEGRATION_EXAMPLES
  exit 1
fi

MISSING_SKAFFOLD_YAMLs="false"
# to whitelist an example from having a skaffold.yaml, add a `grep -v "example_name"` here
cat $EXAMPLES | grep -v "compose" | while read e; do
  NUM_SKAFFOLD_YAML=$(find "examples/$e" -type f -name skaffold.yaml | wc -l)
  if [[ NUM_SKAFFOLD_YAML -eq 0 ]]; then
    echo "examples/$e doesn't have a skaffold.yaml!"
    MISSING_SKAFFOLD_YAMLs="true"
  fi
done

if [[ "$MISSING_SKAFFOLD_YAMLs" == "true" ]]; then
  exit 1
fi

# /examples should use the latest released version
LATEST_RELEASED="skaffold/$(go run ./hack/versions/cmd/latest_released/version.go)"

EXIT_CODE=0

for EXAMPLE in $(find examples -name skaffold*.yaml); do
    if [ "1" != "$(grep -c "apiVersion: ${LATEST_RELEASED}" "${EXAMPLE}")" ]; then
        echo "skaffold version in ${EXAMPLE} should be ${LATEST_RELEASED}"
        EXIT_CODE=1
    fi
done

# /integration/examples should use the latest (even if not released) version
LATEST="skaffold/$(go run ./hack/versions/cmd/latest/version.go)"

for EXAMPLE in $(find integration/examples -name skaffold*.yaml); do
    if [ "1" != "$(grep -c "apiVersion: ${LATEST}" "${EXAMPLE}")" ]; then
        echo "skaffold version in ${EXAMPLE} should be ${LATEST}"
        EXIT_CODE=1
    fi
done

exit ${EXIT_CODE}
