#!/usr/bin/env bash

set -euo pipefail

cd "$(dirname "${0}")"

docker build -t "my-stack/build:1.0" -t "my-stack/run:1.0" .
pack create-builder --no-pull "my-stack/builder:1.0" --builder-config "builder.toml"
