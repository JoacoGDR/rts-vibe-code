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
  type = string
}

variable "instance_class" {
  type    = string
  default = "db.t4g.micro"
}

variable "subnet_ids" {
  type    = list(string)
  default = []
}

variable "vpc_security_group_ids" {
  type    = list(string)
  default = []
}

# Phase 7 will create the DB subnet group, parameter group, and aws_db_instance.

output "endpoint" {
  value       = null
  description = "Filled in by Phase 7"
}
