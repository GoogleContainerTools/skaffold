# Docsy theme for Hugo
FROM alpine:3.9 as docsy
WORKDIR /docsy
RUN apk add --no-cache git
RUN git clone https://github.com/google/docsy.git . \
    && git reset --hard a7141a2eac26cb598b707cab87d224f9105c315d \
    && git submodule update --init --recursive

# Download Hugo binary
FROM alpine:3.9 as download-hugo
ENV HUGO_VERSION=0.51
RUN wget -O- https://github.com/gohugoio/hugo/releases/download/v${HUGO_VERSION}/hugo_extended_${HUGO_VERSION}_Linux-64bit.tar.gz | tar xz

# Hugo base
FROM gcr.io/gcp-runtimes/node_8_5_0_ubuntu_16_0_4 as hugo-base
ENV BASE_URL=http://localhost:1313
WORKDIR /docs
COPY --from=download-hugo /hugo /usr/local/bin/
COPY --from=docsy /docsy ./themes/docsy

# Hugo preview
FROM hugo-base as hugo-preview
EXPOSE 1313
CMD hugo serve --enableGitInfo=false --bind=0.0.0.0 --baseURL=${BASE_URL}

# Hugo build
FROM hugo-base as hugo-build
RUN npm install postcss-cli autoprefixer
COPY docs .
RUN hugo --enableGitInfo=false --baseURL=${BASE_URL}

# Firebase
FROM hugo-base as hugo-publish
RUN npm install -g firebase-tools
COPY --from=hugo-build /docs/public /public
