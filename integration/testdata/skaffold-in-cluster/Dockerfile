FROM gcr.io/gcp-runtimes/ubuntu_16_0_4

COPY ./out/skaffold /usr/bin/skaffold
COPY ./out/skaffold /app/out/skaffold
COPY ./integration/testdata/skaffold-in-cluster /app/integration/testdata/skaffold-in-cluster

ENV KUBECTL_VERSION v1.12.8
ENV KUBECTL_URL https://storage.googleapis.com/kubernetes-release/release/${KUBECTL_VERSION}/bin/linux/amd64/kubectl
RUN curl -O "${KUBECTL_URL}"
RUN chmod +x kubectl
RUN mv kubectl /usr/bin/kubectl

WORKDIR /app

CMD ["skaffold","build", "-f", "./integration/testdata/skaffold-in-cluster/skaffold.yaml", "--cache-artifacts=false"]
