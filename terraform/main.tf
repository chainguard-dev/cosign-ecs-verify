//ECR repo from previous post storing the container
data "aws_ecr_repository" "ecr" {
  name = "distroless-base"
}

//Key used to sign container
data "aws_kms_alias" "cosign" {
  name = "alias/${var.name}"
}

data "aws_caller_identity" "current" {}

data "aws_region" "current" {}