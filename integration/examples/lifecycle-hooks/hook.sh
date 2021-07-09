#!/bin/sh

cat << EOF
Build specification:
    DefaultRepo:    $DEFAULT_REPO
    RPCPort:        $RPC_PORT
    HTTPPort:       $HTTP_PORT
    WorkDir:        $WORK_DIR
    Image:          $IMAGE
    PushImage:      $PUSH_IMAGE
    ImageRepo:      $IMAGE_REPO
    ImageTag:       $IMAGE_TAG
    BuildContext:   $BUILD_CONTEXT
EOF

