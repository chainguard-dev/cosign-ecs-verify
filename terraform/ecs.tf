resource "aws_kms_key" "example" {
  description             = "${var.name}-kms"
  deletion_window_in_days = 7
}

resource "aws_cloudwatch_log_group" "example" {
  name = "${var.name}-log"
}

resource "aws_ecs_cluster" "example" {
  name               = "${var.name}-cluster"
  capacity_providers = ["FARGATE"]

  configuration {
    execute_command_configuration {
      kms_key_id = aws_kms_key.example.arn
      logging    = "OVERRIDE"

      log_configuration {
        cloud_watch_encryption_enabled = true
        cloud_watch_log_group_name     = aws_cloudwatch_log_group.example.name
      }
    }
  }
}

resource "aws_iam_role" "example" {
  name = "${var.name}-ecs-role"

  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ecs.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    },{
      "Effect": "Allow",
      "Principal": {
        "Service": "ecs-tasks.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "example" {
  name = "${var.name}-ecs-policy"
  role = aws_iam_role.example.id

  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecs:*"
      ],
      "Resource": [
        "${aws_ecs_cluster.example.arn}"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "ecr:*"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ecr:*"
      ],
      "Resource": [
        "${data.aws_ecr_repository.ecr.arn}"
      ]
    }
  ]
}
EOF
}

resource "aws_ecs_task_definition" "signed" {
  family                   = "${var.name}-task-definition-signed"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  task_role_arn            = aws_iam_role.example.arn
  execution_role_arn       = aws_iam_role.example.arn
  cpu                      = 1024
  memory                   = 2048
  container_definitions = jsonencode([
    {
      name      = "${var.name}-container"
      image     = var.image_url_signed
      cpu       = 10
      memory    = 512
      essential = true
    }
  ])
}

resource "aws_ecs_task_definition" "unsigned" {
  family                   = "${var.name}-task-definition-unsigned"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  task_role_arn            = aws_iam_role.example.arn
  execution_role_arn       = aws_iam_role.example.arn
  cpu                      = 1024
  memory                   = 2048
  container_definitions = jsonencode([
    {
      name      = "${var.name}-container"
      image     = var.image_url_unsigned
      cpu       = 10
      memory    = 512
      essential = true
    }
  ])
}
