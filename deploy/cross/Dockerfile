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

FROM dockercore/golang-cross:1.13.14 as base

# The base image is not yet available for go 1.14.
# Let's just replace the Go that's installed with a newer one.
RUN rm -Rf /usr/local/go && mkdir /usr/local/go
RUN curl --fail --show-error --silent --location https://dl.google.com/go/go1.14.2.linux-amd64.tar.gz \
    | tar xz --directory=/usr/local/go --strip-components=1

# Cross compile Skaffold for Linux, Windows and MacOS
ARG GOOS
ARG GOARCH
ARG TAGS
ARG LDFLAGS

WORKDIR /skaffold
COPY . ./
RUN if [ "$GOOS" = "darwin" ]; then export CC=o64-clang CXX=o64-clang++; fi; \
    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=1 \
    go build -tags "${TAGS}" -ldflags "${LDFLAGS}" -o /build/skaffold ./cmd/skaffold