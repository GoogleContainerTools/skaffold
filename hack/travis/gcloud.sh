set -e

export CLOUDSDK_CORE_DISABLE_PROMPTS=1
export PROJECT_ID=k8s-skaffold
export CLUSTER=

if [ ! -d "$HOME/google-cloud-sdk/bin" ]; then 
    rm -rf $HOME/google-cloud-sdk
    curl https://sdk.cloud.google.com | bash
fi

source /home/travis/google-cloud-sdk/path.bash.inc

gcloud components update kubectl
gcloud auth activate-service-account --key-file "${GOOGLE_APPLICATION_CREDENTIALS}"
gcloud config set project ${PROJECT_ID}
gcloud container clusters get-credentials ${CLUSTER}
gcloud beta auth configure-docker