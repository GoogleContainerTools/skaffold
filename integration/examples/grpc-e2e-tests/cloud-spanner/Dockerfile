FROM google/cloud-sdk

EXPOSE 9020

RUN gcloud config configurations create emulator
RUN gcloud config set auth/disable_credentials true
RUN gcloud config set api_endpoint_overrides/spanner http://localhost:9020/

ENTRYPOINT ["gcloud", "beta", "emulators", "spanner", "start"]
