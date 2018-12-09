#!/usr/bin/env bash

## This script builds the Skaffold site assuming it's ran from a
## cloned skaffold repo with no submodules initialized. The script initializes the git submodules for
## the site's theme in a standard manner, thus this script can be used locally as well as for the PR review flow.
set -x

readonly DOCSY_COMMIT=$(git config -f .gitmodules submodule.docsy.commit)
readonly REPO_DIR=$(pwd)
readonly BASE_URL=${1:-"http://localhost:1313"}

git submodule init && \
git submodule update --init && \
cd  docs/themes/docsy && \
git checkout ${DOCSY_COMMIT} && \
git submodule update --init --recursive && \
cd  ${REPO_DIR}/docs && \
npm i -D autoprefixer && \
hugo --baseURL=${BASE_URL}
