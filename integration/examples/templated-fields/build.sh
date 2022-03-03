#!/usr/bin/env bash
set -e

if ! [ -x "$(go env GOPATH)/bin/ko" ]; then
    pushd $(mktemp -d)
    curl -L https://github.com/google/ko/archive/v0.9.3.tar.gz | tar --strip-components 1 -zx
    go build -o $(go env GOPATH)/bin/ko .
    popd
fi

output=$($(go env GOPATH)/bin/ko publish --local --preserve-import-paths --tags= . | tee)
ref=$(echo "$output" | tail -n1)

img="${IMAGE_REPO}:${IMAGE_TAG}"
docker tag $ref $img
if $PUSH_IMAGE; then
    docker push $img
fi
