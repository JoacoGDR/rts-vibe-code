# Terraform

Single-region AWS layout (Phase 7 target). Modules live in `modules/` and per-
environment configurations in `envs/`. Phase 0 ships only skeleton files so
`terraform fmt`/`validate` runs in CI without provisioning anything.

```
terraform/
  modules/
    network/       VPC, subnets, NAT, route tables
    eks/           EKS cluster + managed node group
    rds/           RDS Postgres (single AZ for MVP)
    s3-frontend/   S3 bucket + CloudFront for the SPA
  envs/
    dev/           Wires modules together for the dev account
```

To enable a real `terraform apply` in Phase 7:
1. Create an S3 backend bucket and DynamoDB lock table.
2. Fill in `envs/dev/backend.hcl` and uncomment the backend block in
   `envs/dev/main.tf`.
3. Plug in the OIDC IAM role in `.github/workflows/terraform.yml`.
