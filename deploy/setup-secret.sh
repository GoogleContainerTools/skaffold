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

set -x
# set default project id
PROJECT_ID="k8s-skaffold"
METRICS_PROJECT_ID="skaffold-metrics"
KEY_FILE="./secrets/keys.json"
BUCKET_ID="k8s-skaffold-secrets"
LATEST_GCS_PATH="keys.json"

while getopts "p:" opt; do
  case "$opt" in
    p) PROJECT_ID=$OPTARG
    ;;
esac
done

function download_existing_key() {
  # Download a valid key created within the past two weeks.
  KEY_IDS=$(gcloud iam service-accounts keys list --iam-account=metrics-writer@${METRICS_PROJECT_ID}.iam.gserviceaccount.com --project=${METRICS_PROJECT_ID} --managed-by=user --format="value(name)")
  while read -r KEY_ID
  do
    if gsutil cp gs://${BUCKET_ID}/${KEY_ID}.json ${KEY_FILE}; then
      echo "Downloaded existing key to ${KEY_FILE}"
      return 0
    fi
  done <<< "$KEY_IDS"
  return 1
}

function upload_new_key() {
  echo "Creating new service account key..."
  gcloud iam service-accounts keys create ${KEY_FILE} --iam-account=metrics-writer@${METRICS_PROJECT_ID}.iam.gserviceaccount.com --project=${METRICS_PROJECT_ID}
  retVal=$?
  if [ $retVal -ne 0 ]; then
    echo "No key created."
    return 1
  fi
  echo "New service account key created."
  KEY_ID=$(gcloud iam service-accounts keys list --iam-account=metrics-writer@${METRICS_PROJECT_ID}.iam.gserviceaccount.com --project=${METRICS_PROJECT_ID} --managed-by=user --format="value(name)" --limit=1)
  gsutil cp ${KEY_FILE} gs://${BUCKET_ID}/${KEY_ID}.json
  gsutil cp ${KEY_FILE} gs://${BUCKET_ID}/${LATEST_GCS_PATH}
  echo "New service account key uploaded to GCS."
  return 0
}

download_existing_key || upload_new_key
