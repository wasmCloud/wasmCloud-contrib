# Linode API token for the Linode Terraform provider, for creating one please see:
# https://techdocs.akamai.com/linode-api/reference/get-started#personal-access-tokens
variable "akamai_api_token" {}

# This is the size used for the Linode Kubernetes Engine nodes, feel free
# to pick a different size here if another one better suits your needs.
variable "lke_instance_type" {
    default = "g6-dedicated-2"
}

# Region to deploy the cluster in, currently defaults to Dallas, TX
variable "lke_region" {
    default = "us-central"
}