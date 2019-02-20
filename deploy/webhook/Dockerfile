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


FROM gcr.io/gcp-runtimes/ubuntu_16_0_4 as runtime_deps

ENV KUBECTL_VERSION v1.12.0
RUN curl -Lo /usr/local/bin/kubectl https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl  && \
    chmod +x /usr/local/bin/kubectl

ENV HUGO_VERSION=0.54.0
RUN curl -LO https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_extended_${HUGO_VERSION}_Linux-64bit.tar.gz && \
    tar -xzf hugo_extended_${HUGO_VERSION}_Linux-64bit.tar.gz && \
    mv hugo /usr/local/bin/hugo

    # required for nodejs
RUN curl -sL https://deb.nodesource.com/setup_8.x | bash - &&  \
    apt-get update && \
    # latest git is required for hugo and submodule management
    apt-get install -y software-properties-common && \
    add-apt-repository -y ppa:git-core/ppa && \
    apt-get update && \
    apt-get install -y git \
                       # required for hugo, firebase
                       nodejs && \
                       # required for hugo
    npm install -g     postcss-cli \
                       # required for releasing the docs
                       firebase-tools

FROM golang:1.10 as webhook
WORKDIR $GOPATH/src/github.com/GoogleContainerTools/skaffold
COPY . .
RUN go build -o /webhook webhook/webhook.go

FROM runtime_deps
COPY --from=webhook /webhook /webhook
