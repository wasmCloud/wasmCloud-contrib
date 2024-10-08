#!/bin/bash
set -e -x -o pipefail

kind create cluster --config=deploy/kind/cluster.yaml

#############
# Ingress
#############
kubectl apply -k deploy/kind

#############
# Nats
#############
# prefix with helm chart seems to break things down.
# workaround by helm templating instead of kustomize:
kubectl apply -f deploy/base/charts-rendered/nats.yaml

# wait or nats to be ready
kubectl rollout status deploy,sts -l app.kubernetes.io/instance=nats

# test nats:
kubectl exec -it deployment/nats-box -- nats pub test hi

##############################
# wasmCloud operator and wadm
##############################
# we apply Ingress at this step, validate it's ready
kubectl wait --namespace ingress-nginx \
  --for=condition=ready pod \
  --selector=app.kubernetes.io/component=controller \
  --timeout=90s

kustomize build --enable-helm ./deploy/dev | kubectl apply -f -

# validate wasmcloud-operator
kubectl rollout status deploy -l app.kubernetes.io/name=wasmcloud-operator

# validate operator added OAM CRD
kubectl wait --for condition=available apiservices.apiregistration.k8s.io v1beta1.core.oam.dev

# Configure wasmCloud host
kustomize build ./deploy/dev/hosts | kubectl apply -f -
kubectl wait --for condition=established --timeout=60s crd/wasmcloudhostconfigs.k8s.wasmcloud.dev

echo "Validate wadm deployment"
kubectl rollout status deploy -l app.kubernetes.io/instance=wadm

##############################
# Deploy Go and Rust apps
##############################
kustomize build ./deploy/dev/apps | kubectl apply -f -

# wait for http service
kubectl wait --for=jsonpath='{.status.loadBalancer.ingress}' ing/rust-hello-world

curl localhost/rust
