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

FROM golang:1.15 AS generate-files
RUN apt-get update && apt-get install -y unzip moreutils

WORKDIR /protoc
ENV PROTOC_VERSION=3.17.3
RUN plat=$(case $(uname -s)-$(uname -m) in Linux-x86_64) echo linux-x86_64;; Linux-aarch64) echo linux-aarch_64;; *) echo UNKNOWN;; esac); \
  wget -O protoc.zip https://github.com/google/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-${plat}.zip
RUN unzip protoc.zip
ENV PATH="/protoc/bin:${PATH}"

WORKDIR /grpc-gateway
RUN wget -q -O- https://github.com/grpc-ecosystem/grpc-gateway/tarball/v2.5.0 | tar --strip-components 1 -zx

WORKDIR /tmp
ENV GOPROXY=https://proxy.golang.org
ENV GO111MODULE=on

RUN go get \
    github.com/golang/protobuf/protoc-gen-go@v1.5.2 \
    github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.5.0 \
    github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger@v1.7.0 \
    github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@v1.5.0 \
    google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1.0


WORKDIR /proto/google/api
COPY google/api/*.proto ./

# Generate proto files for common enums
WORKDIR /proto/enums
COPY enums/enums.proto enums/markdown.tmpl ./
RUN protoc \
  -I . \
  -I /proto/google/api \
  --grpc-gateway_out=logtostderr=true:. \
  --go_out=plugins=grpc:. \
  --doc_out=. \
  --doc_opt=./markdown.tmpl,enums.md \
  *.proto

# Generate proto files for v1 API
WORKDIR /proto
COPY v1/skaffold.proto v1/markdown.tmpl v1/
RUN protoc \
  -I . \
  -I enums/ \
  -I /protoc/include \
  -I /proto/google/api \
  --grpc-gateway_out=logtostderr=true:. \
  --go_out=. \
  --doc_out=v1/ \
  --doc_opt=v1/markdown.tmpl,v1/index.md \
  --swagger_out=logtostderr=true:. \
  --go-grpc_out=. \
  --go-grpc_opt=require_unimplemented_servers=false \
  v1/*.proto

# Generate proto files for v2 API
WORKDIR /proto
COPY v2/skaffold.proto v2/markdown.tmpl v2/
RUN protoc \
  -I . \
  -I enums/ \
  -I /protoc/include \
  -I /proto/google/api \
  --grpc-gateway_out=logtostderr=true:. \
  --go_out=. \
  --doc_out=v2/ \
  --doc_opt=v2/markdown.tmpl,v2/index.md \
  --swagger_out=logtostderr=true:. \
  --go-grpc_out=. \
  --go-grpc_opt=require_unimplemented_servers=false \
  v2/*.proto

# this is a hack - seemingly grpc-gateway-swagger-gen is sometimes generating titles when they should be descriptions
RUN wget -O jq https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64 && chmod +x ./jq
RUN ./jq 'walk(if type == "object" and has("title") then .description = ([.title, .description] | map(values) | join ("\n")) | del(.title) else .  end)' v1/skaffold.swagger.json | sponge v1/skaffold.swagger.json
RUN ./jq 'walk(if type == "object" and has("title") then .description = ([.title, .description] | map(values) | join ("\n")) | del(.title) else .  end)' v2/skaffold.swagger.json | sponge v2/skaffold.swagger.json

# Append enum docs content to main docs content
RUN cat enums/enums.md >> v1/index.md
RUN cat enums/enums.md >> v2/index.md

# Compare the proto files with the existing proto files
FROM generate-files AS compare
WORKDIR /compare/v1
COPY v1/*.go ./
COPY --from=generate-files /proto/v1/index.md ./

WORKDIR /compare/v2
COPY v2/*.go ./
COPY --from=generate-files /proto/v2/index.md ./
CMD cmp /proto/v1/skaffold.pb.go /compare/v1/skaffold.pb.go && cmp /proto/v1/skaffold.pb.gw.go /compare/v1/skaffold.pb.gw.go && \
    cmp /proto/v2/skaffold.pb.go /compare/v2/skaffold.pb.go && cmp /proto/v2/skaffold.pb.gw.go /compare/v2/skaffold.pb.gw.go
