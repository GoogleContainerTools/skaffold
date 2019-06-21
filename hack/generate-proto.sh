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

export CONTAINER_NAME=generate-proto 
docker build -t $CONTAINER_NAME -f hack/Dockerfile_proto --target generateFiles .
docker run $CONTAINER_NAME cat /pkg/skaffold/server/proto/skaffold.pb.go > pkg/skaffold/server/proto/skaffold.pb.go
docker run $CONTAINER_NAME cat /pkg/skaffold/server/proto/skaffold.pb.gw.go > pkg/skaffold/server/proto/skaffold.pb.gw.go

printf "\nFinished generating proto files, please commit the results.\n"
