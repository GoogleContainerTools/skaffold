#!/bin/bash

bazel build //:skaffold_example.tar
TAR_PATH=$(bazel info bazel-bin)
docker load -i $TAR_PATH/skaffold_example.tar


images=$(echo $IMAGES | tr " " "\n")

for image in $images
do
    docker tag bazel:skaffold_example $image

    if $PUSH_IMAGE
    then
        docker push $image
    fi

done

