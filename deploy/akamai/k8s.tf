provider "helm" {
  kubernetes {
    config_path = local_file.kubeconfig.filename
  }
}

provider "kubernetes" {
  config_path = local_file.kubeconfig.filename
}

# Deploy the nats helm chart in the cluster for wasmCloud to use
resource "helm_release" "nats" {
  name       = "nats"

  repository = "https://nats-io.github.io/k8s/helm/charts/"
  chart      = "nats"

  values = [
    "${file("./k8s/helm-nats-values.yml")}"
  ]
}

# Deploy the wasmCloud operator to enable the integration between Kubernetes and wasmCloud
# This allows you to:
# * Deploy wasmCloud hosts using the WasmCloudHostConfig CRD
# * Deploy and manage wasmCloud applications via `kubectl apply`
resource "helm_release" "wasmcloud_operator" {
  name = "wasmcloud-operator"
  repository = "oci://ghcr.io/wasmcloud/charts"
  chart = "wasmcloud-operator"
  version = "0.1.5"
  wait = true
}

# Deploy the wasmCloud application deployment manager to manage wasmCloud applications
resource "helm_release" "wadm" {
  name       = "wadm"

  repository = "oci://ghcr.io/wasmcloud/charts"
  chart      = "wadm"
  version    = "0.2.4"
  wait       = false

  values = [
    "${file("./k8s/helm-wadm-values.yml")}"
  ]
}

# Deploys a set of wasmCloud hosts in the cluster
resource "kubernetes_manifest" "wasmcloud_host" {
  manifest = yamldecode(file(abspath("${path.module}/k8s/wasmcloud-host.yml")))
}