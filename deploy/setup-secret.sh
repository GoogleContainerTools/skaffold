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
KEY_FILE="./secrets/keys.json"
BUCKET_ID="k8s-skaffold-secrets"
LATEST_GCS_PATH="keys.json"

while getopts "p:" opt; do
  case "$opt" in
    p) PROJECT_ID=$OPTARG
    ;;
esac
done


# create a new valid key
KEY_ID=$(gcloud iam service-accounts keys list --iam-account=metrics-writer@k8s-skaffold.iam.gserviceaccount.com --project=k8s-skaffold --managed-by=user --filter="validAfterTime.date('%Y-%m-%d', Z) = `date +%F`" --format="value(name)" --limit=1)
if [ -z "$KEY_ID" ]; then
  gcloud iam service-accounts keys create ${KEY_FILE} --iam-account=metrics-writer@${PROJECT_ID}.iam.gserviceaccount.com --project=${PROJECT_ID}
  retVal=$?
  if [ $retVal -ne 0 ]; then
    echo "No key created."
    exit 1
  fi
  KEY_ID=$(gcloud iam service-accounts keys list --iam-account=metrics-writer@k8s-skaffold.iam.gserviceaccount.com --project=k8s-skaffold --managed-by=user --filter="validAfterTime.date('%Y-%m-%d', Z) = `date +%F`" --format="value(name)" --limit=1)
fi
gsutil cp ${KEY_FILE} gs://${BUCKET_ID}/${KEY_ID}.json
gsutil cp ${KEY_FILE} gs://${BUCKET_ID}/${LATEST_GCS_PATH}
