#!/bin/bash
# Copyright 2021 The Skaffold Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script creates a github issue if it hasn't been created when there
# are vulnerabilities found in the LTS image.

set -xeo pipefail

# Variables that will be substituted in cloudbuild.yaml.
if [ -z "$_OS_VULN_LABEL" ]; then
  _OS_VULN_LABEL="lts os vuln"
fi
if [ -z "$_REPO" ]; then
  _REPO="GoogleContainerTools/skaffold"
fi

TITLE_OS="LTS image has OS vulnerability!"
OS_VULN_FILE=/workspace/os_vuln.txt

check_existing_issue() {
  label=$1
  # Returns the open issues. There should be only one issue opened at a time.
  issue_num=$(gh issue list --label="$label" --repo="$_REPO" --json number | grep -oP 'number":\s*\K\d+' | head -n 1)

  if [ "$issue_num" ]; then
    echo >&2 "There is already an issue opened for the detected vulnerabilities in the LTS images." && echo "$issue_num"
  else
    echo "-1"
  fi
}

create_issue() {
  title="$1"
  body_file="$2"
  label="$3"
  gh issue create --title="${title}" --label="${label}" --body-file="$body_file" --repo="$_REPO"
}

update_issue() {
  num="$1"
  body_file="$2"
  gh issue edit "$num" --body-file="$body_file" --repo="$_REPO"
}

gh auth login --with-token <token.txt
issue_num=$(check_existing_issue "$_OS_VULN_LABEL")
if [ "$issue_num" -eq "-1" ]; then
  echo "Creating an issue..."
  create_issue "$TITLE_OS" "$OS_VULN_FILE" "$_OS_VULN_LABEL"
else
  echo "Updating issue: #""$issue_num"
  update_issue "$issue_num" "$OS_VULN_FILE"
fi
