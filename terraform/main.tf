data "aws_region" "current" {}

resource "aws_default_subnet" "default" {
  availability_zone = "${data.aws_region.current.name}a"
}

output "cluster_arn" {
  value = aws_ecs_cluster.example.arn
}

output "unsigned_task_arn" {
  value = aws_ecs_task_definition.unsigned.arn
}

output "signed_task_arn" {
  value = aws_ecs_task_definition.signed.arn
}

output "subnet_id" {
  value = aws_default_subnet.default.id
}
