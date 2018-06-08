#!/usr/bin/env bash

# Copyright 2018 The Skaffold Authors
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

# a simple little utility to list PRs for release notes:
# Run ./hack/release_notes.sh --help for more info

go get github.com/google/go-github/github
go build -o out/listpullreqs ./hack/release_note/listpullreqs.go
chmod +x out/listpullreqs
out/listpullreqs $@
