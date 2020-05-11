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

set -e

docker build -t gen-proto -f hack/proto/Dockerfile --target generate-files proto
docker run --rm gen-proto cat skaffold.pb.go > proto/skaffold.pb.go
docker run --rm gen-proto cat skaffold.pb.gw.go > proto/skaffold.pb.gw.go
docker run --rm gen-proto cat index.md > docs/content/en/docs/references/api/grpc.md
docker run --rm gen-proto cat skaffold.swagger.json > docs/content/en/api/skaffold.swagger.json

printf "\nFinished generating proto files, please commit the results.\n"
