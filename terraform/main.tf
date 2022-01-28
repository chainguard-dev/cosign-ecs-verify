//ECR repo from previous post storing the container
data "aws_ecr_repository" "ecr" {
  name = "distroless-base"
}

data "aws_caller_identity" "current" {}

data "aws_region" "current" {}


