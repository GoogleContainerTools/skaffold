#!/usr/bin/env bash
set -e

if ! [ -x "$(command -v ko)" ]; then
    pushd $(mktemp -d)
    curl -L https://github.com/google/ko/archive/v0.8.3.tar.gz | tar --strip-components 1 -zx
    go build -o $(go env GOPATH)/bin/ko .
    popd
fi

output=$($(go env GOPATH)/bin/ko publish --local --preserve-import-paths --tags= . | tee)
ref=$(echo "$output" | tail -n1)

docker tag "$ref" "$IMAGE"
if [[ "${PUSH_IMAGE}" == "true" ]]; then
    docker push "$IMAGE"
fi
