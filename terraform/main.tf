//ECR repo from previous post storing the container
data "aws_ecr_repository" "ecr" {
  name = "distroless-base"
}

data "aws_region" "current" {}

output "cluster_arn" {
  value = aws_ecs_cluster.example.arn
}

output "unsigned_task_arn" {
  value = aws_ecs_task_definition.unsigned.arn
}

output "signed_task_arn" {
  value = aws_ecs_task_definition.signed.arn
}
