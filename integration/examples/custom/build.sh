#!/bin/bash

bazel build //:skaffold_example.tar
docker load -i /usr/local/google/home/priyawadhwa/.cache/bazel/_bazel_priyawadhwa/70bf527ef4c26d952e28ad531f67ba5f/execroot/skaffold/bazel-out/k8-fastbuild/bin/skaffold_example.tar
docker tag bazel:skaffold_example $IMAGE_NAME
docker push $IMAGE_NAME
