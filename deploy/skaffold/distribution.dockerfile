# Copyright 2019 The Skaffold Authors All rights reserved.
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

FROM golang:1.12 as build-skaffold

ARG VERSION
WORKDIR /skaffold
COPY . .
RUN make clean && make out/skaffold-linux-amd64 VERSION=$VERSION && mv out/skaffold-linux-amd64 /usr/bin/skaffold
RUN skaffold credits -d /THIRD_PARTY_NOTICES

FROM gcr.io/gcp-runtimes/ubuntu_16_0_4

RUN apt-get update && \
    apt-get install --no-install-recommends --no-install-suggests -y \
    git python unzip && \
    rm -rf /var/lib/apt/lists/*
COPY --from=docker:18.09.6 /usr/local/bin/docker /usr/local/bin/

WORKDIR /tmp

# Download kubectl
ENV KUBECTL_VERSION v1.12.8
ENV KUBECTL_URL https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
RUN curl -sSfLo kubectl "${KUBECTL_URL}" && \
    chmod +x kubectl && \
    mv kubectl /usr/local/bin/ && \
    rm -rf /tmp/*

# Download helm
ENV HELM_VERSION v2.12.0
ENV HELM_URL https://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz
RUN curl -sSfLo helm.tar.gz "${HELM_URL}" && \
    tar -xvf helm.tar.gz --strip-components 1 && \
    mv helm /usr/local/bin/ && \
    rm -rf /tmp/*

# Download kustomize
ENV KUSTOMIZE_VERSION 2.1.0
ENV KUSTOMIZE_URL https://github.com/kubernetes-sigs/kustomize/releases/download/v${KUSTOMIZE_VERSION}/kustomize_${KUSTOMIZE_VERSION}_linux_amd64
RUN curl -sSfLo kustomize "${KUSTOMIZE_URL}" && \
    chmod +x kustomize && \
    mv kustomize /usr/local/bin/ && \
    rm -rf /tmp/*

# Download kompose
ENV KOMPOSE_VERSION v1.18.0
ENV KOMPOSE_URL https://github.com/kubernetes/kompose/releases/download/${KOMPOSE_VERSION}/kompose-linux-amd64
RUN curl -sSfLo kompose "${KOMPOSE_URL}" && \
    chmod +x kompose && \
    mv kompose /usr/local/bin/ && \
    rm -rf /tmp/*

# Download container-structure-test
ENV CONTAINER_STRUCTURE_TEST_VERSION v1.5.0
ENV CONTAINER_STRUCTURE_TEST_URL https://storage.googleapis.com/container-structure-test/${CONTAINER_STRUCTURE_TEST_VERSION}/container-structure-test-linux-amd64
RUN curl -sSfLo container-structure-test "${CONTAINER_STRUCTURE_TEST_URL}" && \
    chmod +x container-structure-test && \
    mv container-structure-test /usr/local/bin/ && \
    rm -rf /tmp/*

# Download kind
ENV KIND_VERSION v0.6.0
ENV KIND_URL https://github.com/kubernetes-sigs/kind/releases/download/${KIND_VERSION}/kind-linux-amd64
RUN curl -sSfLo kind "${KIND_URL}" && \
    chmod +x kind && \
    mv kind /usr/local/bin/ && \
    rm -rf /tmp/*

# Download bazel
ENV BAZEL_VERSION 0.27.0
ENV BAZEL_URL https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-linux-x86_64
RUN curl -sSfLo bazel "${BAZEL_URL}" && \
    chmod +x bazel && \
    mv bazel /usr/local/bin/ && \
    rm -rf /tmp/*
RUN bazel version

# Download pack
ENV PACK_VERSION 0.4.1
ENV PACK_URL https://github.com/buildpack/pack/releases/download/v${PACK_VERSION}/pack-v${PACK_VERSION}-linux.tgz
RUN curl -sSfLo pack.tgz "${PACK_URL}" && \
    tar -zxf pack.tgz && \
    mv pack /usr/local/bin/ && \
    rm -rf /tmp/*

# Download gcloud
ENV GCLOUD_VERSION 245.0.0
ENV GCLOUD_URL https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-${GCLOUD_VERSION}-linux-x86_64.tar.gz
RUN curl -sSfLo gcloud.tar.gz "${GCLOUD_URL}" && \
    tar -zxf gcloud.tar.gz && \
    mv google-cloud-sdk / && \
    rm -rf /tmp/*
RUN CLOUDSDK_PYTHON="python2.7" /google-cloud-sdk/install.sh \
    --usage-reporting=false \
    --bash-completion=false \
    --disable-installation-options
ENV PATH=$PATH:/google-cloud-sdk/bin
RUN gcloud auth configure-docker

# Distribution specific
COPY --from=build-skaffold /skaffold/out/skaffold-linux-amd64 /usr/bin/skaffold
COPY --from=build-skaffold /THIRD_PARTY_NOTICES /THIRD_PARTY_NOTICES
