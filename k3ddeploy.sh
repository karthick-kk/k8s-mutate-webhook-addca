#!/bin/bash
# Script to prepare and deploy the webhook
set -uo errexit

clusterCA=$(kubectl config view --raw --minify --flatten -o jsonpath='{.clusters[].cluster.certificate-authority-data}')
sed -i "s/caBundle:.*/caBundle: $clusterCA/" deploy/webhook.yaml
k3dserver=$(kubectl get nodes | grep control-plane|head -1|awk '{print $1}')
sed -i "s/export K3D_SERVER_NAME=.*/export K3D_SERVER_NAME=$k3dserver/" ssl/ssl.sh
cd ssl && bash ssl.sh && cd ..
IMAGE_NAME=$(uuidgen)
sed -i "s/image:.*/image: ttl.sh\/$IMAGE_NAME:1h/" deploy/webhook.yaml
docker build -t ttl.sh/${IMAGE_NAME}:1h .
docker push ttl.sh/${IMAGE_NAME}:1h
kubectl apply -f deploy/webhook.yaml




