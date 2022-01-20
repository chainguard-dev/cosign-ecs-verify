resource "aws_kms_key" "example" {
  description             = "${var.name}-kms"
  deletion_window_in_days = 7
}

resource "aws_cloudwatch_log_group" "example" {
  name = "${var.name}-log"
}

resource "aws_ecs_cluster" "example" {
  name = "${var.name}-cluster"

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
        "ecr:GetAuthorizationToken",
        "ecr:Describe*",
        "ecr:List*"
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
    },
    {
      "Effect": "Allow",
      "Action": [
        "kms:*"
      ],
      "Resource": [
        "${data.aws_kms_alias.cosign.arn}"
      ]
    }
  ]
}
EOF
}


resource "aws_ecs_service" "example" {
  name            = "${var.name}-service"
  cluster         = aws_ecs_cluster.example.id
  task_definition = aws_ecs_task_definition.example.arn
  desired_count   = 1
}


resource "aws_ecs_task_definition" "example" {
  family        = "service"
  network_mode  = "none"
  task_role_arn = aws_iam_role.example.arn
  container_definitions = jsonencode([
    {
      name      = "${var.name}-container"
      image     = "${data.aws_caller_identity.current.account_id}.dkr.ecr.${data.aws_region.current.name}.amazonaws.com/${var.image_name}:${var.image_version}"
      cpu       = 10
      memory    = 512
      essential = true
    }
  ])
}