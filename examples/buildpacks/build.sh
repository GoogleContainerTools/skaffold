#!/bin/bash
set -e
image=$(echo $IMAGE)

if [ !-z "$image" ]; then
  pack build $image
  if $PUSH_IMAGE
  then
    docker push $image
  fi
fi
