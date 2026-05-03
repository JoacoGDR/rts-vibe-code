terraform {
  required_version = ">= 1.5.0"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = ">= 5.0"
    }
  }
}

variable "bucket_name" {
  type = string
}

variable "domain_name" {
  type    = string
  default = null
}

# Phase 7 will create:
#   - S3 bucket (private, OAC)
#   - CloudFront distribution with the bucket as origin
#   - ACM certificate (us-east-1) and Route53 record
#   - SPA-friendly default behaviour: 200 -> index.html for /api/* fallback

output "bucket_name" {
  value = var.bucket_name
}

output "distribution_id" {
  value       = null
  description = "Filled in by Phase 7"
}
