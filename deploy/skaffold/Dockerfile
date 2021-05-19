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

# This base image has to be updated manually after running `make build_deps`
ARG GCP_PROJECT=k8s-skaffold
FROM gcr.io/${GCP_PROJECT}/build_deps:94fd5697ee13dc647a21760cb6105ad2fe7510a5 as builder
WORKDIR /skaffold
COPY . .

FROM builder as release
ARG VERSION
ARG USER=nonroot
RUN make clean out/skaffold VERSION=$VERSION && mv out/skaffold /usr/bin/skaffold
RUN rm -rf secrets $SECRET cmd/skaffold/app/cmd/statik/statik.go
RUN skaffold credits -d /THIRD_PARTY_NOTICES 
# RUN useradd -ms /bin/bash $USER
# USER $USER
# WORKDIR /home/$USER
