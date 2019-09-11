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


# This check checks whether the PR compared to master contains any changes
# in the config.go files under pkg/skaffold/schema. If yes, it checks if those changes
# are structural changes or not.
# If they are structural changes and the package is not "latest",
# then we'll fail the PR as we assume anything else than latest is already released and
# shouldn't be changed.
# If the change is latest and it is released, we fail the PR for the same reason.
# If the change is in latest and it is not released yet, it is fine to make changes.


function changeDetected() {
    echo "--------"
    echo "Structural change detected in a released config: $1"
    echo "Please create a new PR first with a new version."
    echo "You can use 'hack/new_version.sh' to generate the new config version."
    echo "Admin rights are required to merge this PR!"
    echo "--------"
    git diff master -- $1
}

set +x
CHANGED_CONFIG_FILES=$(git diff --name-only master -- pkg/skaffold/schema | grep config.go)

if [[ -z "${CHANGED_CONFIG_FILES}" ]]; then
    exit 0
fi

result=0

for f in ${CHANGED_CONFIG_FILES}
do
    cat ${f} > /tmp/a.go
    git show master:${f} > /tmp/b.go
    go run hack/versions/cmd/diff/diff.go -- /tmp/a.go /tmp/b.go > /dev/null 2>&1
    status=$?
    if [[ ${status} -ne 0 ]]; then
        # changes in latest
        if [[ "${f}" == *"latest"* ]]; then
            echo "structural changes in latest config, checking on Github if latest is released..."
            latest=$(go run hack/versions/cmd/latest/latest.go)
            echo ${latest}
            if [[ "${latest}" == *"is released"* ]]; then
                changeDetected ${f}
                result=1
            fi
        else
            changeDetected ${f}
            result=1
        fi
    fi
done

exit $result
