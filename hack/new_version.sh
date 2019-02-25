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

sed -i docs/config.toml -e "s;$CURRENT_VERSION;$NEW_VERSION;g"

cp -R pkg/skaffold/schema/latest pkg/skaffold/schema/${CURRENT_VERSION}

sed -i pkg/skaffold/schema/${CURRENT_VERSION}/*.go -e "s;package latest;package $CURRENT_VERSION;g"

sed pkg/skaffold/schema/${PREV_VERSION}/upgrade_test.go -e "s;$CURRENT_VERSION;$NEW_VERSION;g" > pkg/skaffold/schema/${CURRENT_VERSION}/upgrade_test.go
sed -i pkg/skaffold/schema/${CURRENT_VERSION}/upgrade_test.go -e "s;$PREV_VERSION;$CURRENT_VERSION;g"

sed pkg/skaffold/schema/${PREV_VERSION}/upgrade.go -e "s;$CURRENT_VERSION;$NEW_VERSION;g" > pkg/skaffold/schema/${CURRENT_VERSION}/upgrade.go
sed -i pkg/skaffold/schema/${CURRENT_VERSION}/upgrade.go -e "s;$PREV_VERSION;$CURRENT_VERSION;g"

sed -i pkg/skaffold/schema/${PREV_VERSION}/upgrade*.go -e "s;latest;$CURRENT_VERSION;g"
goimports -w pkg/skaffold/schema/${PREV_VERSION}/upgrade*.go

sed -i pkg/skaffold/schema/latest/config.go -e "s;$CURRENT_VERSION;$NEW_VERSION;g"

find integration -name "skaffold.yaml" -print0 | xargs -0 -I xx sed -i xx -e "s;$CURRENT_VERSION;$NEW_VERSION;g"

sed pkg/skaffold/schema/versions.go -i -e "s;\(.*\)$PREV_VERSION.Version\(.*\)$PREV_VERSION\(.*\);&\n\1$CURRENT_VERSION.Version\2$CURRENT_VERSION\3;g"
sed pkg/skaffold/schema/versions.go -i -e "s;\(.*\)/$PREV_VERSION\(.*\);&\n\1/$CURRENT_VERSION\2;g"

make generate-schemas

git --no-pager diff --minimal

make test

echo
echo "---------------------------------------"
echo
echo "Files generated for $NEW_VERSION. Don't worry about the hack/check-docs change failure, it is expected!"
echo "Other tests should have passed. For the docs change, commit the results and rerun 'make test'."
echo "Please double check manually the generated files as well: the upgrade functionality, and all the examples:"
echo
git status -s
echo
echo "---------------------------------------"
