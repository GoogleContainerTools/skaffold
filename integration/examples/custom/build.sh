#!/bin/bash

bazel build //:skaffold_example.tar
TAR_PATH=$(bazel info bazel-bin)
docker load -i $TAR_PATH/skaffold_example.tar
docker tag bazel:skaffold_example $IMAGE_NAME
docker push $IMAGE_NAME
