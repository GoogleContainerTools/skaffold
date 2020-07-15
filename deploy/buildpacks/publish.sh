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

PACK_VERSION=v0.12.0

docker build . --build-arg PACK_VERSION=${PACK_VERSION} -t gcr.io/k8s-skaffold/pack:${PACK_VERSION} -t gcr.io/k8s-skaffold/pack:latest
docker push gcr.io/k8s-skaffold/pack:${PACK_VERSION}
docker push gcr.io/k8s-skaffold/pack:latest
