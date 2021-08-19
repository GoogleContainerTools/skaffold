PROJECT_NAME=$(gcloud config get-value project)
REGION=us-central1
ZONE=a

# Create GKE cluster
gcloud config set project $PROJECT_NAME
gcloud config set compute/zone $REGION-$ZONE
gcloud config set compute/region $REGION
gcloud services enable container.googleapis.com compute.googleapis.com
gcloud compute networks create $PROJECT_NAME-network \
  --subnet-mode=custom
gcloud compute networks subnets create $PROJECT_NAME-subnet \
  --network=$PROJECT_NAME-network \
  --range=10.0.0.0/24
gcloud container clusters create $PROJECT_NAME-cluster \
  --zone "$REGION-$ZONE" \
  --machine-type "n1-standard-1" \
  --disk-size "10" \
  --num-nodes "1" \
  --enable-ip-alias \
  --network "projects/$PROJECT_NAME/global/networks/$PROJECT_NAME-network" \
  --subnetwork "projects/$PROJECT_NAME/regions/$REGION/subnetworks/$PROJECT_NAME-subnet" \
  --node-locations "$REGION-$ZONE"

# Install latest version of Skaffold
curl -Lo skaffold https://storage.googleapis.com/skaffold/releases/latest/skaffold-linux-amd64 && \
sudo install skaffold /usr/local/bin/
rm ./skaffold