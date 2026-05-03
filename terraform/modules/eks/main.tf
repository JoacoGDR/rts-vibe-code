terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

variable "cluster_name" {
  type = string
}

variable "vpc_id" {
  type    = string
  default = null
}

variable "subnet_ids" {
  type    = list(string)
  default = []
}

variable "kubernetes_version" {
  type    = string
  default = "1.30"
}

# Phase 7 will instantiate the official terraform-aws-modules/eks module here.

output "cluster_name" {
  value = var.cluster_name
}
