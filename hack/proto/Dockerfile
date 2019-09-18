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

# Install protoc
FROM alpine:3.10 AS protoc
RUN apk add --no-cache unzip

ENV PROTOC_VERSION=3.8.0
RUN wget -O protoc.zip https://github.com/google/protobuf/releases/download/v${PROTOC_VERSION}/protoc-${PROTOC_VERSION}-linux-x86_64.zip
RUN unzip protoc.zip -d protoc

# Install protoc-gen-go and grpc-gateway binaries; then generate files.
FROM golang:1.12.9-alpine3.10 AS generateFiles
RUN apk add --no-cache protobuf=3.6.1-r1 git

COPY --from=protoc /protoc/include /usr/local/include

ENV GIT_TAG=v1.2.0
RUN go get -d -u github.com/golang/protobuf/protoc-gen-go
RUN git -C "$(go env GOPATH)"/src/github.com/golang/protobuf checkout $GIT_TAG
RUN go install github.com/golang/protobuf/protoc-gen-go

WORKDIR /go/src/github.com/grpc-ecosystem
RUN git clone https://github.com/grpc-ecosystem/grpc-gateway.git

ENV GRPC_GATEWAY_REVISION=e6f18d33a7b3bfa5b94f3d5fb513252184ce2d90
WORKDIR /go/src/github.com/grpc-ecosystem/grpc-gateway
RUN git checkout ${GRPC_GATEWAY_REVISION} && git checkout -b v${GRPC_GATEWAY_REVISION}

ENV GO111MODULE=on
RUN go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-grpc-gateway
RUN go get -u github.com/grpc-ecosystem/grpc-gateway/protoc-gen-swagger
RUN go get -u github.com/golang/protobuf/protoc-gen-go

WORKDIR /proto
COPY skaffold.proto ./
RUN protoc -I /usr/local/include -I . -I $GOPATH/src -I $GOPATH/src/github.com/grpc-ecosystem/grpc-gateway/third_party/googleapis --grpc-gateway_out=logtostderr=true:. skaffold.proto --go_out=plugins=grpc:.

# Compare the proto files with the existing proto files
FROM generateFiles AS compare
WORKDIR /compare
COPY *.go ./
CMD cmp /proto/skaffold.pb.go skaffold.pb.go && cmp /proto/skaffold.pb.gw.go skaffold.pb.gw.go
