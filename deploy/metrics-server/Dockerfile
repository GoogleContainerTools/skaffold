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

# Use the offical Golang image to create a build artifact.
# This is based on Debian and sets the GOPATH to /go.
# https://hub.docker.com/_/golang

FROM gcr.io/gcp-runtimes/ubuntu_16_0_4 as runtime_deps
RUN apt-get update && \
  apt-get install --no-install-recommends --no-install-suggests -y \
  git python unzip && \
  rm -rf /var/lib/apt/lists/*
COPY --from=docker:18.09.6 /usr/local/bin/docker /usr/local/bin/

FROM runtime_deps as builder
RUN apt-get update && apt-get install --no-install-recommends --no-install-suggests -y \
  ca-certificates \
  curl \
  build-essential  && \
  rm -rf /var/lib/apt/lists/*
COPY --from=golang:1.12 /usr/local/go /usr/local/go
ENV PATH /usr/local/go/bin:/go/bin:$PATH
WORKDIR /skaffold
COPY . .
RUN make out/metrics-server && mv out/metrics-server /usr/bin/metrics-server

# Use a Docker multi-stage build to create a lean production image.
FROM runtime_deps as distribution
# Copy the binary to the production image from the builder stage.
COPY --from=builder /usr/bin/metrics-server /metrics-server
ENV SKAFFOLD_SURVEY_BUCKET "test_skaffold_survey"


# Run the web service on container startup.
CMD ["/metrics-server"]