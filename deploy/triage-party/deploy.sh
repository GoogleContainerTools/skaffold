#!/bin/bash

# Copyright 2021 Google Inc.
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

set -eux

# Export this environment variable before running this script
echo "token path: ${GITHUB_TOKEN_PATH}"

export PROJECT=k8s-skaffold
IMAGE="gcr.io/k8s-skaffold/teaparty:$(date +%F-%s)"
export
export SERVICE_NAME=skaffold-triage-party
export CONFIG_FILE=config/examples/skaffold.yaml

docker build -t "${IMAGE}" --build-arg "CFG=${CONFIG_FILE}" .

docker push "${IMAGE}" || exit 2

readonly token="$(cat "${GITHUB_TOKEN_PATH}")"
gcloud beta run deploy "${SERVICE_NAME}" \
    --project "${PROJECT}" \
    --image "${IMAGE}" \
    --set-env-vars="GITHUB_TOKEN=${token},PERSIST_BACKEND=cloudsql,PERSIST_PATH=host=k8s-skaffold:us-central1:triage-party2 user=postgres password=${DB_PASS}" \
    --allow-unauthenticated \
    --region us-central1 \
    --memory 384Mi \
    --platform managed