#!/usr/bin/env bash
#
# Copyright 2023 Google LLC
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

set -eEufo pipefail

# The kubectl context
context=${SKAFFOLD_KUBE_CONTEXT:-$(kubectl config current-context)}

# The Kubernetes resource to port forward to, e.g., "service/myapp".
resource=$1

# The namespace of the resource.
namespace=${2:-$SKAFFOLD_NAMESPACE}

# Port forwarding ports.
local_port=${3:-4503}
remote_port=${4:-8080}

# Seconds to wait for port forwarding to be ready, used if `nc` is unavailable.
sleep_secs=${5:-5}

kubectl port-forward \
  --context "$context" \
  --namespace "$namespace" \
  "$resource" \
  "$local_port:$remote_port" \
  > /dev/null \
  &
port_forward_pid=$!

# Stop `kubectl port-forward`` when the script exits.
trap 'kill "$port_forward_pid"' EXIT

# Wait for port forwarding to be ready.
# Use the `nc` command if available, otherwise `sleep`
if command -v nc > /dev/null 2>&1; then
  while ! nc -z localhost "$local_port" > /dev/null 2>&1; do
    sleep 0.1
  done
else
  # `nc` is unavailable, sleep for a number of seconds.
  sleep "$sleep_secs"
fi

# The integration test.
curl --silent "localhost:$local_port"
curl --silent "localhost:$local_port"
curl --silent "localhost:$local_port"
curl --silent "localhost:$local_port"
curl --silent "localhost:$local_port"
curl --silent "localhost:$local_port"
curl --silent "localhost:$local_port"
