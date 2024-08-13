terraform {
  required_providers {
    linode = {
      source  = "linode/linode"
      version = "2.25.0"
    }
  }
}

# Configure the Linode Provider
provider "linode" {
  token = var.akamai_api_token
}

# Create an LKE cluster for running the wasmCloud cluster
resource "linode_lke_cluster" "wasmcloud" {
    label       = "wasmcloud"
    k8s_version = "1.30"
    region      = var.lke_region
    tags        = ["wasmcloud"]

    pool {
        type  = var.lke_instance_type
        count = 3
    }
}

# This is the kubeconfig you can use for interacting with the LKE cluster,
# please note that it's needed by the commands listed in the helm.tf file for
# the purposes of deploying the wasmCloud cluster inside of LKE.
resource "local_file" "kubeconfig" {
  content  = base64decode(linode_lke_cluster.wasmcloud.kubeconfig)
  filename = "${path.module}/kubeconfig"
}

# From here on, k8s.tf will take over the 