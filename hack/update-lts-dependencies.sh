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

set -euo pipefail
# NOTE: This scripts expects be run from skaffold root "skaffold/" NOT "skaffold/hack"
# TODO script should likely not write files to where script is done but most likely should write to tmp or out/
# TODO script should fetch sha values from remote endpoints vs generate them

ARCH=amd64
DOCKERFILE_DIR="deploy/skaffold"
DIGESTS_DIR="${DOCKERFILE_DIR}/digests"

KUBECTL_REPO="kubernetes/kubernetes"
LATEST_KUBECTL_VERSION=$(curl --silent "https://api.github.com/repos/$KUBECTL_REPO/releases/latest" | jq -r .tag_name)
HELM_REPO="helm/helm"
LATEST_HELM_VERSION=$(curl --silent "https://api.github.com/repos/$HELM_REPO/releases/latest" | jq -r .tag_name)
KUSTOMIZE_REPO="kubernetes-sigs/kustomize"
LATEST_KUSTOMIZE_VERSION=$(curl --silent "https://api.github.com/repos/$KUSTOMIZE_REPO/releases/latest" | jq -r .tag_name)
LATEST_KUSTOMIZE_VERSION=${LATEST_KUSTOMIZE_VERSION#kustomize/v}; #Remove "kustomize/v" prefix
KPT_REPO="GoogleContainerTools/kpt"
LATEST_KPT_VERSION=$(curl --silent "https://api.github.com/repos/GoogleContainerTools/kpt/tags" | jq -r .[0].name)
LATEST_KPT_VERSION=${LATEST_KPT_VERSION#v}; #Remove "v" prefix
LATEST_GCLOUD_VERSION=$(gcloud version --format json| jq -r '."Google Cloud SDK"')

for dockerfile in "Dockerfile.deps.lts"; do
  OLD_KUBECTL_VERSION=$(sed -n "s/^.*ENV KUBECTL_VERSION \s*\(\S*\).*$/\1/p" ${DOCKERFILE_DIR}/${dockerfile})
  OLD_HELM_VERSION=$(sed -n "s/^.*ENV HELM_VERSION \s*\(\S*\).*$/\1/p" ${DOCKERFILE_DIR}/${dockerfile})
  OLD_KUSTOMIZE_VERSION=$(sed -n "s/^.*ENV KUSTOMIZE_VERSION \s*\(\S*\).*$/\1/p" ${DOCKERFILE_DIR}/${dockerfile})
  OLD_KPT_VERSION=$(sed -n "s/^.*ENV KPT_VERSION \s*\(\S*\).*$/\1/p" ${DOCKERFILE_DIR}/${dockerfile})
  OLD_GCLOUD_VERSION=$(sed -n "s/^.*ENV GCLOUD_VERSION \s*\(\S*\).*$/\1/p" ${DOCKERFILE_DIR}/${dockerfile})
done 


read -e -i "$LATEST_KUBECTL_VERSION" -p "Enter version to upgrade kubectl to: " KUBECTL_VERSION
name="${KUBECTL_VERSION:-$LATEST_KUBECTL_VERSION}"
read -e -i "$LATEST_HELM_VERSION" -p "Enter version to upgrade helm to: " HELM_VERSION
name="${HELM_VERSION:-$LATEST_HELM_VERSION}"
read -e -i "$LATEST_KUSTOMIZE_VERSION" -p "Enter version to upgrade kustomize to: " KUSTOMIZE_VERSION
name="${KUSTOMIZE_VERSION:-$LATEST_KUSTOMIZE_VERSION}"
read -e -i "$LATEST_KPT_VERSION" -p "Enter version to upgrade kpt to: " KPT_VERSION
name="${KPT_VERSION:-$LATEST_KPT_VERSION}"
read -e -i "$LATEST_GCLOUD_VERSION" -p "Enter version to upgrade gcloud to: " GCLOUD_VERSION
name="${GCLOUD_VERSION:-$LATEST_GCLOUD_VERSION}"


KUBECTL_URL=https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/${ARCH}/kubectl
echo "Updating kubectl to version: $KUBECTL_VERSION..."
wget -q -O  kubectl "${KUBECTL_URL}"

# take the shasum and put into correct digests/ dir
sha512sum kubectl > ${DIGESTS_DIR}/kubectl.amd64.sha512

echo "Updating helm to version: $HELM_VERSION..."
HELM_URL=https://get.helm.sh/helm-${HELM_VERSION}-linux-${ARCH}.tar.gz
wget -q -O  helm.tar.gz "${HELM_URL}"

# upload the binary to skaffold gcs bucket
gsutil -q cp helm.tar.gz gs://skaffold/deps/helm/helm-${HELM_VERSION}-linux-amd64.tar.gz

# take the shasum and put into correct digests/ dir
sha256sum helm.tar.gz > ${DIGESTS_DIR}/helm.amd64.sha256

echo "Updating kustomize to version: $KUSTOMIZE_VERSION..."
KUSTOMIZE_URL=https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize/v${KUSTOMIZE_VERSION}/kustomize_v${KUSTOMIZE_VERSION}_linux_${ARCH}.tar.gz
wget -q -O  kustomize.tar.gz "${KUSTOMIZE_URL}"

# upload the binary to skaffold gcs bucket
gsutil -q cp kustomize.tar.gz gs://skaffold/deps/kustomize/v${KUSTOMIZE_VERSION}/kustomize_v${KUSTOMIZE_VERSION}_linux_amd64.tar.gz

# take the shasum and put into correct digests/ dir
sha256sum kustomize.tar.gz > ${DIGESTS_DIR}/kustomize.amd64.sha256

echo "Updating kpt to version: $KPT_VERSION..."
KPT_URL=https://github.com/GoogleContainerTools/kpt/releases/download/v${KPT_VERSION}/kpt_linux_amd64
wget -q -O  kpt "${KPT_URL}"

# upload the binary to skaffold gcs bucket
gsutil -q cp kpt gs://skaffold/deps/kpt/v${KPT_VERSION}/kpt_linux_amd64

# take the shasum and put into correct digests/ dir
sha256sum kpt > ${DIGESTS_DIR}/kpt.amd64.sha256

echo "Updating gcloud to version: $GCLOUD_VERSION..."
GCLOUD_URL=https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-${GCLOUD_VERSION}-linux-x86_64.tar.gz
wget -q -O  gcloud.tar.gz $GCLOUD_URL

# take the shasum and put into correct digests/ dir
sha256sum gcloud.tar.gz > ${DIGESTS_DIR}/gcloud.amd64.sha256

echo ""
echo "Updated kubectl to version: $KUBECTL_VERSION (from: $OLD_KUBECTL_VERSION)"
echo "Updated helm to version: $HELM_VERSION (from: $OLD_HELM_VERSION)"
echo "Updated kustomize to version: $KUSTOMIZE_VERSION (from: $OLD_KUSTOMIZE_VERSION)"
echo "Updated kpt to version: $KPT_VERSION (from: $OLD_KPT_VERSION))"
echo "Updated gcloud to version: $GCLOUD_VERSION (from: $OLD_GCLOUD_VERSION)"

echo ""
echo "WARNING: the method used to get deps comes from combination of Github latest release and latest git tag on a repo. \
These methods of getting the latest version of a binary are not error prone and should be manually checked.  \
For gcloud there is no api endpoint to check for the latest version so the output of 'gcloud version' is used asssuming \
the machine running this command has the latest gcloud version.  Manual modification of the script is required in cases where \
the version found by the script is incorrect."

for dockerfile in "Dockerfile.deps" "Dockerfile.deps.lts" "Dockerfile.deps.slim"; do
    sed -i "s/ENV KUBECTL_VERSION .*/ENV KUBECTL_VERSION ${KUBECTL_VERSION}/" ${DOCKERFILE_DIR}/${dockerfile}
    sed -i "s/ENV HELM_VERSION .*/ENV HELM_VERSION ${HELM_VERSION}/" ${DOCKERFILE_DIR}/${dockerfile}
    sed -i "s/ENV KUSTOMIZE_VERSION .*/ENV KUSTOMIZE_VERSION ${KUSTOMIZE_VERSION}/" ${DOCKERFILE_DIR}/${dockerfile}
    sed -i "s/ENV KPT_VERSION .*/ENV KPT_VERSION ${KPT_VERSION}/" ${DOCKERFILE_DIR}/${dockerfile}
    sed -i "s/ENV GCLOUD_VERSION .*/ENV GCLOUD_VERSION ${GCLOUD_VERSION}/" ${DOCKERFILE_DIR}/${dockerfile}
done 

for artifact in "gcloud.tar.gz" "helm.tar.gz" "kpt" "kubectl" "kustomize.tar.gz"; do
  rm ${artifact}
done
