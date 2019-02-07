# Copyright 2019 The Skaffold Authors All rights reserved.
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

# Docsy theme for Hugo
FROM alpine:3.9 as docsy
WORKDIR /docsy
RUN apk add --no-cache git
RUN git clone https://github.com/google/docsy.git . \
    && git reset --hard a7141a2eac26cb598b707cab87d224f9105c315d \
    && git submodule update --init --recursive

# Download Hugo binary
FROM alpine:3.9 as download-hugo
ENV HUGO_VERSION 0.54.0
ENV HUGO_URL https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_extended_${HUGO_VERSION}_Linux-64bit.tar.gz
RUN wget -O- "${HUGO_URL}" | tar xz

# Hugo base
FROM node:8.15.0-stretch as hugo-base
RUN npm install -g firebase-tools
WORKDIR /docs
RUN npm install postcss-cli autoprefixer
COPY --from=download-hugo /hugo /bin/
COPY --from=docsy /docsy ./themes/docsy

# Hugo preview
FROM hugo-base as hugo-preview
ENV BASE_URL=http://localhost:1313
EXPOSE 1313
CMD hugo serve --enableGitInfo=false --bind=0.0.0.0 --baseURL=${BASE_URL}

# Hugo publish
FROM hugo-base as hugo-publish
ENV HUGO_ENV production
ENV BASE_URL=https://skaffold-latest.firebaseapp.com
COPY . ./
RUN hugo --enableGitInfo=false --baseURL=${BASE_URL}
CMD firebase deploy --only hosting:head --project ${PROJECT_ID}
