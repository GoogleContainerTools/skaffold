#!/bin/bash

# Copyright 2019 The Skaffold Authors
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

# Changes the current directory to where Kokoro has checked out the GitHub repository.
pushd $KOKORO_ARTIFACTS_DIR/github/skaffold >/dev/null
    # Prevent Jib (Maven/Gradle) from crashing on Kokoro.
    # Kokoro is a "clean" environment and doesn't have a Maven settings file (~/.m2/settings.xml).
    # When Skaffold tries to sync that non-existent file into a Docker container, Docker 
    # mistakenly creates a FOLDER named 'settings.xml' instead. Jib then crashes because 
    # it can't read a folder as a configuration file. 
    # Pointing home to /tmp avoids this file-vs-folder conflict.
    export MAVEN_OPTS="-Duser.home=/tmp"
    export GRADLE_USER_HOME="/tmp/.gradle"
    GCP_ONLY=true GCP_PROJECT=skaffold-ci-cd AR_REGION=us-central1 GKE_REGION=us-central1 make integration-in-docker
popd

