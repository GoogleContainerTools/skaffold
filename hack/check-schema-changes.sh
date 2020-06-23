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


# This check checks whether the PR compared to origin/master contains any changes
# in the config.go files under pkg/skaffold/schema. If yes, it checks if those changes
# are structural changes or not.
# If they are structural changes and the package is not "latest",
# then we'll fail the PR as we assume anything else than latest is already released and
# shouldn't be changed.
# If the change is latest and it is released, we fail the PR for the same reason.
# If the change is in latest and it is not released yet, it is fine to make changes.

go run hack/versions/cmd/schema_check/check.go
