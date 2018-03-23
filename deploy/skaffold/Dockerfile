# Copyright 2018 Google, Inc. All rights reserved.
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

# Second stage is used to install the binaries that we need
FROM gcr.io/gcp-runtimes/ubuntu_16_0_4

RUN apt-get update && \
    apt-get install --no-install-recommends --no-install-suggests -y \
        ca-certificates \
        curl \
        build-essential \
        git \ 
        gcc \
        python-dev \
        python-setuptools \
        apt-transport-https \
        lsb-release

COPY --from=golang:1.10 /usr/local/go /usr/local/go
ENV PATH /usr/local/go/bin:/go/bin:$PATH
ENV GOPATH /go/

ENV KUBECTL_VERSION v1.9.3
RUN curl -Lo /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl  && \
    chmod +x /usr/local/bin/kubectl

ENV HELM_VERSION v2.8.1
RUN curl -LO https://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz && \
    tar -zxvf helm-${HELM_VERSION}-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/local/bin/helm

ENV CLOUD_SDK_VERSION 193.0.0
RUN easy_install -U pip && \
    pip install -U crcmod && \
    export CLOUD_SDK_REPO="cloud-sdk-$(lsb_release -c -s)" && \
    echo "deb https://packages.cloud.google.com/apt $CLOUD_SDK_REPO main" > /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | apt-key add - && \
    apt-get update && apt-get install -y google-cloud-sdk=${CLOUD_SDK_VERSION}-0 $INSTALL_COMPONENTS && \
    gcloud config set core/disable_usage_reporting true && \
    gcloud config set component_manager/disable_update_check true && \
    gcloud config set metrics/environment skaffold_docker_image && \
    gcloud --version

RUN curl -LO https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v1.4.3/docker-credential-gcr_linux_amd64-1.4.3.tar.gz && \
    tar -zxvf docker-credential-gcr_linux_amd64-1.4.3.tar.gz && \
    mv docker-credential-gcr /usr/bin/docker-credential-gcr && \
    rm docker-credential-gcr_linux_amd64-1.4.3.tar.gz && \
    docker-credential-gcr configure-docker

ENV PATH /usr/local/go/bin:/go/bin:/google-cloud-sdk/bin:$PATH

WORKDIR /go/src/github.com/GoogleCloudPlatform/skaffold
COPY . .
RUN make out/skaffold-linux-amd64 && mv out/skaffold-linux-amd64 /usr/bin/skaffold

CMD ["make", "integration"]
