#!/usr/bin/env bash
set -Eefuo pipefail

if ! [ -x "$(go env GOPATH)/bin/ko" ]; then
    pushd "$(mktemp -d)"
    curl -L https://github.com/ko-build/ko/archive/v0.13.0.tar.gz | tar --strip-components 1 -zx
    go build -o "$(go env GOPATH)"/bin/ko .
    popd
fi

output=$("$(go env GOPATH)"/bin/ko publish --local --preserve-import-paths --tags= . | tee)
ref=$(echo "$output" | tail -n1)

docker tag "$ref" "$IMAGE"
if [[ "${PUSH_IMAGE}" == "true" ]]; then
    echo "Pushing $IMAGE"
    docker push "$IMAGE"
else
    echo "Not pushing $IMAGE"
fi
