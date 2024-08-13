#!/bin/bash

# Following practices from https://github.com/kubernetes-sigs/kustomize/blob/master/examples/chart.md

# Pass in -r to remove the charts directory
rm -rf deploy/base/charts

WADM_CHART_VERSION=0.2.5
WASMCLOUD_OPERATOR_CHART_VERSION=0.1.5
NATS_CHART_VERSION=1.2.2

helm pull oci://ghcr.io/wasmcloud/charts/wadm \
    --version $WADM_CHART_VERSION \
    --untar \
    --untardir deploy/base/charts
helm template wadm --values ./deploy/base/wadm-values.yaml --version $WADM_CHART_VERSION ./deploy/base/charts/wadm > deploy/base/charts-rendered/wadm.yaml

helm pull oci://ghcr.io/wasmcloud/charts/wasmcloud-operator \
    --version $WASMCLOUD_OPERATOR_CHART_VERSION \
    --untar \
    --untardir deploy/base/charts
helm template wasmcloud-operator --version $WASMCLOUD_OPERATOR_CHART_VERSION ./deploy/base/charts/wasmcloud-operator > deploy/base/charts-rendered/wasmcloud-operator.yaml

# ugh this should work but gzip header error on the tgz, helm isn't handling redirects
# helm pull https://nats-io.github.io/k8s/helm/charts/ --version ^1.2 \
#     --untar \
#     --untardir deploy/base/charts/nats
# workaround:
helm repo add nats https://nats-io.github.io/k8s/helm/charts/
helm repo update
helm pull nats/nats --version $NATS_CHART_VERSION
tar -xvf nats-${NATS_CHART_VERSION}.tgz -C deploy/base/charts
helm template nats --values ./deploy/base/nats-values.yaml --version $NATS_CHART_VERSION ./deploy/base/charts/nats > deploy/base/charts-rendered/nats.yaml
# cleanup
rm nats-1.2.2.tgz
