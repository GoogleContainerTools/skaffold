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

echo "Collecting schemas"
cp -R docs-v2/content/en/schemas "fs/assets/schemas_generated"


if [[ -d ${SECRET} ]]; then
   echo "generating embedded files for secret"
   cp -R ${SECRET} "fs/assets/secrets_generated"
fi

if [ -n "${FIRELOG_API_KEY_FILE+x}" ] && [ -f ${FIRELOG_API_KEY_FILE} ]; then
   echo "generating embedded files for firelog API key"
   mkdir -p "fs/assets/firelog_generated"
   cp ${FIRELOG_API_KEY_FILE} "fs/assets/firelog_generated/key.txt"
fi

echo "Used for marking generating embedded files task complete, don't modify this." > fs/assets/check.txt
