#!/usr/bin/env bash


## This script starts a preview of the Skaffold site assuming it's ran from a
## cloned skaffold repo with no submodules initialized. The script initializes the git submodules for
## the site's theme in a standard manner, thus this script can be used locally as well as for the PR review flow.
set -x

readonly REPO_DIR=$(pwd)
readonly BASE_URL=${1:-"http://localhost:1313"}

bash ${REPO_DIR}/deploy/docs/build.sh ${BASE_URL}

cd ${REPO_DIR}/docs

hugo serve --bind=0.0.0.0 -D --baseURL=${BASE_URL}
