#!/bin/bash

# Following practices from https://github.com/kubernetes-sigs/kustomize/blob/master/examples/chart.md

rm -rf deploy/base/charts

WASMCLOUD_PLATFORM_CHART_VERSION=0.1.0

helm pull oci://ghcr.io/wasmcloud/charts/wasmcloud-platform \
    --version $WASMCLOUD_PLATFORM_CHART_VERSION \
    --untar \
    --untardir deploy/base/charts
helm template wasmcloud-platform --values ./deploy/base/values.yaml --version $WASMCLOUD_PLATFORM_CHART_VERSION ./deploy/base/charts/wasmcloud-platform > deploy/base/charts-rendered/wasmcloud-platform.yaml
