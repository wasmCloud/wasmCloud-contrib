resource "aws_ecs_cluster" "wasmcloud" {
  name = "wasmcloud"
  setting {
    name  = "containerInsights"
    value = "enabled"
  }
}

resource "aws_service_discovery_private_dns_namespace" "wasmcloud" {
  name        = "cluster.wasmcloud"
  description = "wasmcloud"
  vpc         = aws_vpc.wasmcloud.id
}

