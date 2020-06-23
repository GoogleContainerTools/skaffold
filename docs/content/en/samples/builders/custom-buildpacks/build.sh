#!/usr/bin/env bash
set -e

if ! [ -x "$(command -v ko)" ]; then
    GO111MODULE=on go get -mod=readonly github.com/google/ko/cmd/ko@v0.4.0
fi

output=$(ko publish --local --preserve-import-paths --tags= . | tee)
ref=$(echo $output | tail -n1)

docker tag $ref $IMAGE
if $PUSH_IMAGE; then
    docker push $IMAGE
fi
