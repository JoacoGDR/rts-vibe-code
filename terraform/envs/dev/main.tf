terraform {
  required_version = ">= 1.5.0"

  # Backend wired up in Phase 7. For now, local state keeps `terraform validate`
  # passing in CI without external dependencies.
  # backend "s3" {}

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

variable "region" {
  type    = string
  default = "us-east-1"
}

variable "name_prefix" {
  type    = string
  default = "supremacy-dev"
}

module "network" {
  source = "../../modules/network"
  name   = var.name_prefix
}

module "eks" {
  source       = "../../modules/eks"
  cluster_name = "${var.name_prefix}-eks"
  vpc_id       = module.network.vpc_id
  subnet_ids   = module.network.private_subnet_ids
}

module "rds" {
  source     = "../../modules/rds"
  name       = "${var.name_prefix}-pg"
  subnet_ids = module.network.private_subnet_ids
}

module "frontend" {
  source      = "../../modules/s3-frontend"
  bucket_name = "${var.name_prefix}-spa"
}
