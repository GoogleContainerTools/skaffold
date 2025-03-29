#!/usr/bin/env bash

set -e
set -o pipefail

readonly SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "PWD=${PWD}"
echo "SCRIPT_DIR=${SCRIPT_DIR}"

source "$SCRIPT_DIR/1_dependencies.sh"
source "$SCRIPT_DIR/2_create-ppa.sh"
source "$SCRIPT_DIR/3_test-ppa.sh"
source "$SCRIPT_DIR/4_upload-ppa.sh"

echo
echo "++++++++++++++++++++++++++++"
echo "> Installing dependencies..."
echo "++++++++++++++++++++++++++++"
echo
dependencies

echo
echo "++++++++++++++++++++++++++++"
echo "> Creating PPA..."
echo "++++++++++++++++++++++++++++"
echo
create_ppa

echo
echo "++++++++++++++++++++++++++++"
echo "> Testing PPA..."
echo "++++++++++++++++++++++++++++"
echo
test_ppa

echo
echo "++++++++++++++++++++++++++++"
echo "> Uploading PPA..."
echo "++++++++++++++++++++++++++++"
echo
upload_ppa
