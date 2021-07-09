#!/bin/sh

cat << EOF
Build specification:
    DefaultRepo:    $SKAFFOLD_DEFAULT_REPO
    RPCPort:        $SKAFFOLD_RPC_PORT
    HTTPPort:       $SKAFFOLD_HTTP_PORT
    WorkDir:        $SKAFFOLD_WORK_DIR
    Image:          $SKAFFOLD_IMAGE
    PushImage:      $SKAFFOLD_PUSH_IMAGE
    ImageRepo:      $SKAFFOLD_IMAGE_REPO
    ImageTag:       $SKAFFOLD_IMAGE_TAG
    BuildContext:   $SKAFFOLD_BUILD_CONTEXT
EOF

