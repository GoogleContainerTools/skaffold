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

FROM golang:1.13.4-alpine3.10 AS generate-files
RUN apk add --no-cache protobuf=3.6.1-r1 unzip jq moreutils

WORKDIR /protoc
ENV PROTOC_VERSION=3.11.0
RUN wget -O protoc.zip https://github.com/google/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip
RUN unzip protoc.zip

WORKDIR /grpc-gateway
RUN wget -q -O- https://github.com/grpc-ecosystem/grpc-gateway/tarball/v1.9.1 | tar --strip-components 1 -zx

WORKDIR /tmp
ENV GOPROXY=https://proxy.golang.org
ENV GO111MODULE=on
RUN go get \
    github.com/golang/protobuf/protoc-gen-go@v1.3.2 \
    github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway@v1.9.1 \
    github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger@v1.9.1 \
    github.com/pseudomuto/protoc-gen-doc/cmd/protoc-gen-doc@v1.3.2

WORKDIR /proto
COPY skaffold.proto markdown.tmpl ./
RUN protoc \
  -I . \
  -I /protoc/include  \
  -I /grpc-gateway/third_party/googleapis \
  --grpc-gateway_out=logtostderr=true:. \
  --go_out=plugins=grpc:. \
  --doc_out=. \
  --doc_opt=./markdown.tmpl,index.md \
  --swagger_out=logtostderr=true:. \
  *.proto

# this is a hack - seemingly grpc-gateway-swagger-gen is sometimes generating titles when they should be descriptions
RUN jq 'walk(if type == "object" and has("title") then .description = ([.title, .description] | map(values) | join ("\n")) | del(.title) else .  end)' skaffold.swagger.json | sponge skaffold.swagger.json

# Compare the proto files with the existing proto files
FROM generate-files AS compare
WORKDIR /compare
COPY *.go ./
COPY --from=generate-files /proto/index.md ./
CMD cmp /proto/skaffold.pb.go skaffold.pb.go && cmp /proto/skaffold.pb.gw.go skaffold.pb.gw.go
