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
docker run --rm gen-proto cat enums/github.com/GoogleContainerTools/skaffold/proto/enums/enums.pb.go > proto/enums/enums.pb.go

# Copy v1 files
docker run --rm gen-proto cat v1/skaffold.pb.go > proto/v1/skaffold.pb.go
docker run --rm gen-proto cat v1/skaffold.pb.gw.go > proto/v1/skaffold.pb.gw.go

# Copy v2 files
docker run --rm gen-proto cat /proto/github.com/GoogleContainerTools/skaffold/proto/v2/skaffold.pb.go > proto/v2/skaffold.pb.go
docker run --rm gen-proto cat /proto/github.com/GoogleContainerTools/skaffold/proto/v2/skaffold.pb.gw.go > proto/v2/skaffold.pb.gw.go

# Get docs from docker image
docker run --rm gen-proto cat v1/index.md > docs/content/en/docs/references/api/grpc.md
docker run --rm gen-proto cat v1/skaffold.swagger.json > docs/content/en/api/skaffold.swagger.json

printf "\nFinished generating proto files, please commit the results.\n"
