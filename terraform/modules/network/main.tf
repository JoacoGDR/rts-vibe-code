terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

variable "name" {
  type        = string
  description = "Name prefix"
}

variable "cidr_block" {
  type    = string
  default = "10.20.0.0/16"
}

variable "azs" {
  type    = list(string)
  default = ["us-east-1a", "us-east-1b"]
}

# Phase 7 will expand this into VPC, subnets, NAT, IGW, route tables. Phase 0
# leaves it as a documented stub so terraform validate stays green without
# provisioning anything.

output "vpc_id" {
  value       = null
  description = "Filled in once VPC resources are added in Phase 7"
}

output "private_subnet_ids" {
  value       = []
  description = "Filled in once VPC resources are added in Phase 7"
}
