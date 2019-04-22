#!/bin/bash

bazel build //:skaffold_example.tar
docker load -i /private/var/tmp/_bazel_priyawadhwa/84207b180d3c99eceb0884a6401eb421/execroot/skaffold/bazel-out/darwin-fastbuild/bin/skaffold_example.tar
docker tag bazel:skaffold_example $IMAGE_NAME
docker push $IMAGE_NAME
