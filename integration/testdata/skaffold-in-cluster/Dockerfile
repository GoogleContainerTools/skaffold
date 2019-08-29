FROM gcr.io/gcp-runtimes/ubuntu_16_0_4

COPY ./skaffold /usr/bin/skaffold
RUN chmod +x /usr/bin/skaffold
COPY ./test-build /test-build
COPY ./skaffold.yaml ./skaffold.yaml

ENV KUBECTL_VERSION v1.12.8
ENV KUBECTL_URL https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
RUN curl -O "${KUBECTL_URL}"
RUN chmod +x kubectl
RUN mv kubectl /usr/bin/kubectl