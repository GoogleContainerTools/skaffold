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

FROM neilotoole/xcgo:go1.17 as base

# Cross compile Skaffold for Linux, Windows and MacOS
ARG GOOS
ARG GOARCH
ARG TAGS
ARG LDFLAGS
ARG CGO_ENABLED=1

WORKDIR /skaffold
COPY . ./
RUN if [ "$GOOS" = "darwin" ]; then export CC=o64-clang CXX=o64-clang++; fi; \
    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=$CGO_ENABLED \
    go build -tags "${TAGS}" -ldflags "${LDFLAGS}" -o /build/skaffold ./cmd/skaffold
