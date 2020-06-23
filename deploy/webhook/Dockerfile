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

# Download Docsy theme for Hugo
FROM alpine:3.10 as download-docsy
ENV DOCSY_VERSION a7141a2eac26cb598b707cab87d224f9105c315d
ENV DOCSY_URL https://github.com/google/docsy.git
RUN apk add --no-cache git
WORKDIR /docsy
RUN git clone "${DOCSY_URL}" . && \
    git reset --hard "${DOCSY_VERSION}" && \
    git submodule update --init --recursive && \
    rm -rf .git

# Download Hugo
FROM alpine:3.10 as download-hugo
ENV HUGO_VERSION 0.67.1
ENV HUGO_URL https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_extended_${HUGO_VERSION}_Linux-64bit.tar.gz
RUN wget -O- "${HUGO_URL}" | tar xz

# Download kubectl
FROM alpine:3.10 as download-kubectl
ENV KUBECTL_VERSION v1.12.0
ENV KUBECTL_URL https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
RUN wget -O kubectl "${KUBECTL_URL}"
RUN chmod +x kubectl

FROM node:10.15.3-stretch as runtime_deps
ENV FIREBASE_TOOLS_VERSION 7.13.1
RUN npm install -g firebase-tools@${FIREBASE_TOOLS_VERSION} postcss-cli
WORKDIR /app/docs
RUN npm install autoprefixer
COPY --from=download-docsy /docsy ./themes/docsy
COPY --from=download-hugo /hugo /usr/local/bin/
COPY --from=download-kubectl /kubectl /usr/local/bin/

FROM golang:1.14 as webhook
WORKDIR /skaffold
COPY . .
RUN go build -mod=vendor -o /webhook webhook/webhook.go

FROM runtime_deps
COPY --from=webhook /webhook /webhook
