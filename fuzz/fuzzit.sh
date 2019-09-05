#!/bin/bash

# Copyright 2019 The Skaffold Authors
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
set -xe

# Validate arguments
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <fuzz-type>"
    exit 1
fi

# Configure
NAME=skaffold
ROOT=.
TYPE=$1

# Setup
export GO111MODULE=on
go install \
    golang.org/x/net/proxy \
    github.com/dvyukov/go-fuzz/go-fuzz \
    github.com/dvyukov/go-fuzz/go-fuzz-build \
    github.com/fuzzitdev/fuzzit/v2
go mod vendor -v
rm -rf gopath
mkdir -p gopath/src
mv vendor/* gopath/src
rm -rf vendor
export GOPATH=$PWD/gopath
export GO111MODULE=off
fuzzit --version

# Fuzz
function fuzz {
    FUNC=Fuzz$1
    TARGET=$2
    go-fuzz-build -libfuzzer -func $FUNC -o fuzzer.a .
    clang -fsanitize=fuzzer fuzzer.a -o fuzzer
    fuzzit create job --type $TYPE $NAME/$TARGET fuzzer
}
fuzz ParseConfig parse-config
fuzz ParseReference parse-reference
fuzz ServerTCP control-api-tcp
fuzz ServerHTTP control-api-http
