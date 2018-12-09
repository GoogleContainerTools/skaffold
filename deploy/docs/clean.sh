#!/usr/bin/env bash

readonly REPO_DIR=$(pwd)

pushd ${REPO_DIR}/docs

rm -rf public resources node_modules package-lock.json &&  \
git submodule deinit -f . && \
rm -rf themes/docsy/* && \
rm -rf ${REPO_DIR}/.git/modules/docsy

popd
