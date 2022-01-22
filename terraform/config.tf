terraform {
  backend "s3" {
    bucket = "cosign-ecs-verify"
    key    = "cosign-ecs-verify/aws-ecs/terraform_state"
    region = "us-west-2"
  }
}
