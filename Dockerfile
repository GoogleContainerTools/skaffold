# Copyright 2018 The Skaffold Authors All rights reserved.
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

FROM golang:1.10-alpine AS build

RUN apk add --update \
      curl \
      git \
      make \
      py-pip \
      python \
      python-dev \
      && true

RUN mkdir /out
RUN mkdir -p /out/etc/apk && cp -r /etc/apk/* /out/etc/apk/
RUN apk add --no-cache --initdb --root /out \
    alpine-baselayout \
    busybox \
    ca-certificates \
    coreutils \
    git \
    libc6-compat \
    libgcc \
    libstdc++ \
    python \
    && true

ENV SKAFFOLD $GOPATH/src/github.com/GoogleContainerTools/skaffold
RUN mkdir -p "$(dirname ${SKAFFOLD})"
COPY . $SKAFFOLD

WORKDIR $SKAFFOLD
RUN make \
    && cp out/skaffold /out/usr/local/bin/skaffold

WORKDIR /out

RUN ln -s /lib /lib64

ENV KUBECTL_VERSION v1.10.6
RUN curl --silent --location "https://dl.k8s.io/${KUBECTL_VERSION}/bin/linux/amd64/kubectl" --output usr/local/bin/kubectl \
    && chmod +x usr/local/bin/kubectl

ENV DOCKER_VERSION 18.03.0
RUN curl --silent --location "https://download.docker.com/linux/static/stable/x86_64/docker-${DOCKER_VERSION}-ce.tgz" \
    | tar xz docker/docker \
    && mv docker/docker usr/local/bin/docker && rm -rf docker

ENV CLOUD_SDK_VERSION 206.0.0
RUN curl --silent --location "https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-${CLOUD_SDK_VERSION}-linux-x86_64.tar.gz" \
    | tar xz \
    && pip install --root=/out crcmod==1.7

ENV HELM_VERSION v2.9.1
RUN curl --silent --location "https://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz" \
    | tar xz linux-amd64/helm \
    && mv linux-amd64/helm usr/local/bin/helm

ENV DOCKER_CREDENTIAL_GCR_VERSION 1.5.0
RUN curl --silent --location "https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v${DOCKER_CREDENTIAL_GCR_VERSION}/docker-credential-gcr_linux_amd64-${DOCKER_CREDENTIAL_GCR_VERSION}.tar.gz" \
    | tar xz ./docker-credential-gcr \
    && mv docker-credential-gcr usr/local/bin/docker-credential-gcr
# TODO: docker-credential-gcr configure-docker

ENV KUSTOMIZE_VERSION 1.0.6
RUN curl --silent --location "https://github.com/kubernetes-sigs/kustomize/releases/download/v${KUSTOMIZE_VERSION}/kustomize_${KUSTOMIZE_VERSION}_linux_amd64" --output usr/local/bin/kustomize \
    && chmod +x usr/local/bin/kustomize

ENV BAZEL_VERSION 0.16.1
RUN curl --silent --location "https://github.com/bazelbuild/bazel/releases/download/${BAZEL_VERSION}/bazel-${BAZEL_VERSION}-linux-x86_64" --output usr/local/bin/bazel \
    && chmod +x usr/local/bin/bazel

FROM scratch
CMD skaffold
COPY --from=build  /out /
