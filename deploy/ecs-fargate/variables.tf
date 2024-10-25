variable "aws_region" {
  description = "The AWS region things are created in"
  type        = string
  default     = "us-east-1"
}

variable "aws_profile" {
  description = "AWS Profile (if any)"
  type        = string
}

variable "private_az_count" {
  description = "Number of Private AZs to cover in a given region"
  type        = number
  default     = 1
}

variable "public_az_count" {
  description = "Number of Public AZs to cover in a given region"
  type        = number
  default     = 1
}

## NATS
variable "nats_image" {
  description = "NATS.io Docker Image"
  type        = string
  default     = "cgr.dev/chainguard/nats:latest"
}

variable "nats_count" {
  description = "Number of docker containers to run"
  type        = number
  default     = 1
}


variable "nats_cpu" {
  description = "Fargate instance CPU units to provision (1 vCPU = 1024 CPU units)"
  type        = number
  default     = 1024
}

variable "nats_memory" {
  description = "Fargate instance memory to provision (in MiB)"
  type        = number
  default     = 2048
}

variable "nats_allowed_cidrs" {
  description = "CIDR blocks to allow access to NATS LB"
  type        = list(string)
  default     = []
}

variable "nats_enable_ingress" {
  description = "If the stack should configure a load balancer for NATS"
  type        = bool
  default     = true
}

## WADM

variable "wadm_image" {
  description = "WADM Docker Image"
  type        = string
  default     = "ghcr.io/wasmcloud/wadm:v0.17.0-wolfi"
}

variable "wadm_count" {
  description = "Number of docker containers to run"
  type        = number
  default     = 1
}

variable "wadm_cpu" {
  description = "Fargate instance CPU units to provision (1 vCPU = 1024 CPU units)"
  type        = number
  default     = 1024
}

variable "wadm_memory" {
  description = "Fargate instance memory to provision (in MiB)"
  type        = number
  default     = 2048
}

## WASMCLOUD

variable "wasmcloud_image" {
  description = "Wasmcloud Docker Image"
  type        = string
  default     = "ghcr.io/wasmcloud/wasmcloud:1.3.1-wolfi"
}

variable "wasmcloud_workload_min_count" {
  description = "Minimum number of workload tasks to run"
  type        = number
  default     = 1
}

variable "wasmcloud_workload_max_count" {
  description = "Max number of workload tasks to run"
  type        = number
  default     = 1
}

variable "wasmcloud_public_ingress_count" {
  description = "Number of ingress tasks to run"
  type        = number
  default     = 1
}

variable "wasmcloud_cpu" {
  description = "Fargate instance CPU units to provision (1 vCPU = 1024 CPU units)"
  type        = number
  default     = 1024
}

variable "wasmcloud_memory" {
  description = "Fargate instance memory to provision (in MiB)"
  type        = number
  default     = 2048
}

variable "wasmcloud_allowed_cidrs" {
  description = "CIDR blocks to allow access to NATS LB"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "wasmcloud_enable_ingress" {
  description = "If the stack should configure a load balancer for wasmcloud ingress"
  type        = bool
  default     = true
}
