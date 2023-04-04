#!/bin/bash -ex

apt-get update

# Install system deps
apt-get install -y --no-install-recommends \
  mariadb-client-core-10.3 \
  rpm \
  ruby \
  ruby-dev \
  rsyslog \
  build-essential \
  cmake \
  libssl-dev \
  opensc \
  unzip \
  python3-pip \
  gcc \
  ca-certificates \
  openssl \
  softhsm2 \
  pkg-config \
  libtool \
  autoconf \
  automake

PROTO_ARCH=x86_64
if [ "${TARGETPLATFORM}" = linux/arm64 ]
then
  PROTO_ARCH=aarch_64
fi

curl -L https://github.com/google/protobuf/releases/download/v3.20.1/protoc-3.20.1-linux-"${PROTO_ARCH}".zip -o /tmp/protoc.zip
unzip /tmp/protoc.zip -d /usr/local/protoc

# Override default GOBIN and GOCACHE
export GOBIN=/usr/local/bin GOCACHE=/tmp/gocache

# Install protobuf and testing/dev tools.
# Note: The version of golang/protobuf is partially tied to the version of grpc
# used by Boulder overall. Updating it may require updating the grpc version
# and vice versa.
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2.0
go install github.com/rubenv/sql-migrate/...@v1.1.2
go install golang.org/x/tools/cmd/stringer@latest
go install github.com/letsencrypt/pebble/cmd/pebble-challtestsrv@master
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.51.0

go clean -cache
go clean -modcache

pip3 install -r /tmp/requirements.txt

gem install --no-document -v 1.14.2 fpm

apt-get autoremove -y libssl-dev ruby-dev cmake pkg-config libtool autoconf automake
apt-get clean -y

# Tell git to trust the directory where the boulder repo volume is mounted
# by `docker compose`.
git config --global --add safe.directory /boulder

rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
