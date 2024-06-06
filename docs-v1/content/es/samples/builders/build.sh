#!/bin/bash

bazel build //:skaffold_example.tar
TAR_PATH=$(bazel info bazel-bin)
docker load -i $TAR_PATH/skaffold_example.tar

image=$(echo $IMAGE)

if [ ! -z "$image" ]; then
  pack build $image
  if $PUSH_IMAGE
  then
    docker push $image
  fi
fi
