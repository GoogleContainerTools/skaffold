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

# First stage is used to install the binaries that we need
FROM ubuntu
RUN apt-get update && \
    apt-get install --no-install-recommends --no-install-suggests -y \
        curl \
        ca-certificates \
        git

ENV KUBECTL_VERSION v1.9.3
RUN curl -Lo /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl  && \
    chmod +x /usr/local/bin/kubectl

ENV HELM_VERSION v2.8.1
RUN curl -LO https://storage.googleapis.com/kubernetes-helm/helm-${HELM_VERSION}-linux-amd64.tar.gz && \
    tar -zxvf helm-${HELM_VERSION}-linux-amd64.tar.gz && \
    mv linux-amd64/helm /usr/local/bin/helm

# Second stage is to build the skaffold binary from source
FROM golang:1.10
WORKDIR /go/src/github.com/GoogleCloudPlatform/skaffold
COPY . .
RUN make

FROM ubuntu 
WORKDIR /root/
COPY --from=0 /usr/local/bin/kubectl    /usr/local/bin/kubectl
COPY --from=0 /usr/local/bin/helm       /usr/local/bin/helm
COPY --from=0 /usr/bin/git              /usr/bin/git

COPY --from=1 /go/src/github.com/GoogleCloudPlatform/skaffold/out/skaffold .

CMD ["/root/skaffold", "version"]
