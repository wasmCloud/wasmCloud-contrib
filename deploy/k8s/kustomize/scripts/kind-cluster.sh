#!/bin/bash
set -e -x -o pipefail

kind create cluster --config=deploy/kind/cluster.yaml --wait 30s

#############
# Ingress
#############
kubectl apply -k deploy/kind

kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s

######################
# wasmCloud platform
######################
kubectl apply -k ./deploy/dev

# wait for nats
kubectl rollout status deploy,sts -l app.kubernetes.io/name=nats

# wait for wadm
kubectl wait --for=condition=available --timeout=600s deploy -l app.kubernetes.io/name=wadm

# wait for wasmcloud-operator
kubectl wait --for=condition=available --timeout=600s deploy -l app.kubernetes.io/name=wasmcloud-operator

# validate operator added OAM CRD
kubectl wait --for condition=available apiservices.apiregistration.k8s.io v1beta1.core.oam.dev

# Configure wasmCloud host
kubectl apply -k ./deploy/dev/hosts
kubectl wait --for condition=established --timeout=60s crd/wasmcloudhostconfigs.k8s.wasmcloud.dev

##############################
# Deploy apps
############################## 
kustomize build ./deploy/dev/apps | kubectl apply -f -

# wait for http service
kubectl wait --for=jsonpath='{.status.loadBalancer.ingress}' ing/rust-hello-world

curl localhost/rust
