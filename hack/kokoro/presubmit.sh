#!/bin/bash

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

echo "Clamping MSS"
sudo iptables -A FORWARD -p tcp --tcp-flags SYN,RST SYN -j TCPMSS  --clamp-mss-to-pmtu

export DOCKER_NAMESPACE=gcr.io/k8s-skaffold
source $KOKORO_GFILE_DIR/common.sh

pushd $KOKORO_ARTIFACTS_DIR/github/skaffold >/dev/null
    GCP_ONLY=true make integration-in-docker
popd

