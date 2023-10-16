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

# Find the pods created by the Skaffold run.
ns_pods=($(kubectl get pods \
  --all-namespaces \
  --context "$SKAFFOLD_KUBE_CONTEXT" \
  --output go-template='{{range .items}}{{.metadata.namespace}}/{{.metadata.name}}{{"\n"}}{{end}}' \
  --selector "skaffold.dev/run-id=$SKAFFOLD_RUN_ID"))

# Base directory for the coverage profile data files.
reportbasedir=reports/coverage

# Array to collect report directories for all pods.
reportdirs=()

# Copy coverage profile data files from pods to local host.
# Use separate local directories for each pod.
for ns_pod in "${ns_pods[@]}"; do
  reportdir="$reportbasedir/$ns_pod"
  mkdir -p "$reportdir"
  reportdirs+=("$reportdir")
  ns=$(echo "$ns_pod" | cut -d'/' -f1)
  pod=$(echo "$ns_pod" | cut -d'/' -f2)
  kubectl exec \
    --context "$SKAFFOLD_KUBE_CONTEXT" \
    --namespace "$ns" \
    "$pod" \
    -- \
    tar -C /coverage-data -cf - . | tar -xf - -C "$reportdir"
done

# Comma-separated string of report directories for use with `go tool covdata`.
reportdirs_commasep=$(echo "${reportdirs[@]}" | tr " " ",")

# Report the percent statements covered metric per package on the terminal.
go tool covdata percent -i "$reportdirs_commasep"
