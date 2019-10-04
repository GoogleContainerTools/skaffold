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

EXAMPLES=`mktemp`
INTEGRATION_EXAMPLES=`mktemp`

find examples -mindepth 1 -maxdepth 1 -type d -not -empty -exec basename {} \; | sort  > $EXAMPLES
find integration/examples -mindepth 1 -maxdepth 1 -type d -not -empty -exec basename {} \; | sort > $INTEGRATION_EXAMPLES

EXAMPLES_MINUS_INTEGRATION_EXAMPLES=`awk 'FNR==NR{ array[$0];next} {if ( $1 in array ) next; print $1}' $INTEGRATION_EXAMPLES $EXAMPLES`

if [ ! -z "$EXAMPLES_MINUS_INTEGRATION_EXAMPLES" ]; then
  echo "Every code sample that is in ./examples should also be in ./integration/examples"
  echo "The following are in examples but not in integration/examples:"
  echo $EXAMPLES_MINUS_INTEGRATION_EXAMPLES
  exit 1
fi

