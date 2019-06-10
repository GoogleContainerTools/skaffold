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

readonly DOCS_CHANGES=`git diff --name-status master | grep "docs/" | wc -l`

if [ $DOCS_CHANGES -gt 0 ]; then
  echo "There are $DOCS_CHANGES changes in docs, testing site generation..."
  make build-docs-preview
fi