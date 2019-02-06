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

CURRENT_VERSION=`go run hack/new_config_version/version.go`
echo "Current config version: $CURRENT_VERSION"

echo "Please enter new config version:"
read NEW_VERSION

echo "Please enter previous config version:"
read PREV_VERSION

echo "Generating changes for new config version $NEW_VERSION..."

sed -i docs/config.toml -e "s;$CURRENT_VERSION;$NEW_VERSION;"

cp -R pkg/skaffold/schema/latest pkg/skaffold/schema/${CURRENT_VERSION}

sed -i pkg/skaffold/schema/${CURRENT_VERSION}/*.go -e "s;latest;$CURRENT_VERSION;"

sed pkg/skaffold/schema/${PREV_VERSION}/upgrade_test.go -e "s;$CURRENT_VERSION;$NEW_VERSION;" > pkg/skaffold/schema/${CURRENT_VERSION}/upgrade_test.go
sed -i pkg/skaffold/schema/${CURRENT_VERSION}/upgrade_test.go -e "s;$PREV_VERSION;$CURRENT_VERSION;"

sed pkg/skaffold/schema/${PREV_VERSION}/upgrade.go -e "s;$CURRENT_VERSION;$NEW_VERSION;" > pkg/skaffold/schema/${CURRENT_VERSION}/upgrade.go
sed -i pkg/skaffold/schema/${CURRENT_VERSION}/upgrade.go -e "s;$PREV_VERSION;$CURRENT_VERSION;"

sed -i pkg/skaffold/schema/latest/config.go -e "s;$CURRENT_VERSION;$NEW_VERSION;"

find integration -name "skaffold.yaml" | xargs -I xx sed -i xx -e 's;v1beta4;v1beta5;g'

git --no-pager diff --minimal

git status -s

go test ${GOPATH}/src/github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/${CURRENT_VERSION}

echo "Please double check the generated files, the upgrade functionality, and the examples!!"