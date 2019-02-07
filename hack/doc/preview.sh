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
set -e

DOCDIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
SKAFFOLDDIR="${DOCDIR}/../.."

# Build the Hugo image
tar -cz -C ${DOCDIR} Dockerfile | docker build --target hugo-preview -t skaffold-docs-preview -

# Find the local files to mount
mounts="-v ${SKAFFOLDDIR}/docs/config.toml:/docs/config.toml"
for dir in $(find ${SKAFFOLDDIR}/docs -type dir -mindepth 1 -maxdepth 1 | grep -v themes | grep -v public | grep -v resources); do
    mounts="$mounts -v $dir:/docs/$(basename $dir):delegated"
done

# Run Hugo
docker run --rm -ti -p 1313:1313 $mounts skaffold-docs-preview
