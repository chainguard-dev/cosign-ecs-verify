NAME ?= cosign-ecs
KEY_ALIAS ?= ${NAME}-key
AWS_REGION ?= us-west-2
ACCOUNT_ID ?= $(shell aws sts get-caller-identity --query Account --output text)
EVENT ?= event.json
# These have the "signed" and "unsigned" tags, respectively, but we pin the digest.
IMAGE_URL_SIGNED ?= public.ecr.aws/d1r0p2a6/ecs-cosign-demo2@sha256:31267f66e1aeb1f4a301faa75a7222927fa5bf1382697667b32ff580f9c6bfac
IMAGE_URL_UNSIGNED ?= public.ecr.aws/d1r0p2a6/ecs-cosign-demo2@sha256:8648f155b13820cb73521d08ab0cc22a906735e5e932b0dc3e81a06008940e5c

AWS_DEFAULT_REGION = ${AWS_REGION}
STACK_NAME = ${NAME}-stack
SAM_TEMPLATE = template.yml
PACKAGED_TEMPLATE = packaged.yml

GO_SRCS := $(wildcard cosign-ecs-function/*.go)

export AWS_REGION AWS_DEFAULT_REGION

################################################################################
# Terraform
################################################################################

.PHONY: tf_clean tf_init tf_get tf_plan tf_apply tf_fmt tf_destroy tf_refresh

tf_clean:
	cd terraform/ && \
	rm -rf .terraform \
	rm -rf plan.out

tf_init: 
	cd terraform/ && \
	terraform init

tf_get:
	cd terraform/ && \
	terraform get

tf_plan:
	cd terraform/ && \
	terraform plan \
		-var="name=${NAME}" \
		-var="image_url_signed=${IMAGE_URL_SIGNED}" \
		-var="image_url_unsigned=${IMAGE_URL_UNSIGNED}" \
		-out=plan.out

tf_apply:
	cd terraform/ && \
	terraform apply \
		-var="name=${NAME}" \
		-var="image_url_signed=${IMAGE_URL_SIGNED}" \
		-var="image_url_unsigned=${IMAGE_URL_UNSIGNED}" \
		-auto-approve

tf_fmt:
	cd terraform/ && \
	terraform fmt

tf_destroy:
	cd terraform/ && \
	terraform destroy \
		-var="name=${NAME}" \
		-var="image_url_signed=${IMAGE_URL_SIGNED}" \
		-var="image_url_unsigned=${IMAGE_URL_UNSIGNED}" \
		-auto-approve

tf_refresh:
	cd terraform/ && \
	terraform refresh \
		-var="name=${NAME}" \
		-var="image_url_signed=${IMAGE_URL_SIGNED}" \
		-var="image_url_unsigned=${IMAGE_URL_UNSIGNED}"

################################################################################
# SAM
################################################################################

.PHONY: sam_build sam_package sam_deploy sam_local sam_local_debug sam_delete

cosign-ecs-function/cosign-ecs-function: $(GO_SRCS) cosign-ecs-function/go.mod cosign-ecs-function/go.sum
	cd ./cosign-ecs-function && go mod tidy && \
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o cosign-ecs-function .

go_build: cosign-ecs-function/cosign-ecs-function

# `sam build` will build the serverless binary itself
sam_build:
	sam build --cached

sam_package: sam_build $(SAM_TEMPLATE)
	sam package \
		--template-file ${SAM_TEMPLATE} \
		--output-template-file ${PACKAGED_TEMPLATE} \
		--resolve-s3

sam_deploy: sam_package
	KEY_PEM=$$(cat cosign.pub; echo .); var=${var%.}; \
	sam deploy \
		--template-file ${SAM_TEMPLATE} \
		--resolve-s3 \
		--capabilities CAPABILITY_IAM \
		--stack-name ${STACK_NAME} \
		--parameter-overrides \
			"ParameterKey=KeyArn,ParameterValue=''" \
			"ParameterKey=KeyPem,ParameterValue='$${KEY_PEM}'"

sam_local: sam_build
	KEY_PEM=$$(cat cosign.pub; echo .); var=${var%.}; \
	sam local invoke \
		--event ${EVENT} \
		--template ${SAM_TEMPLATE} \
		--parameter-overrides \
			"ParameterKey=KeyArn,ParameterValue=''" \
			"ParameterKey=KeyPem,ParameterValue='$${KEY_PEM}'"

sam_local_debug: sam_build
	KEY_PEM=$$(cat cosign.pub; echo .); var=${var%.}; \
	sam local invoke \
		--event ${EVENT} \
		--template ${SAM_TEMPLATE} \
		--parameter-overrides \
			"ParameterKey=KeyArn,ParameterValue=''" \
			"ParameterKey=KeyPem,ParameterValue='$${KEY_PEM}'" \
		--debug

sam_delete:
	sam delete \
		--stack-name ${STACK_NAME} \
		--region ${AWS_REGION} \
		--no-prompts
#  if --no-prompts, it ignores $AWS_REGION

################################################################################
# Test it out!
################################################################################

.PHONY: run_signed_task run_unsigned_task task_status

run_signed_task:
	TASK_DEF_ARN=$$(cd terraform && terraform output -raw signed_task_arn) && \
	CLUSTER_ARN=$$(cd terraform && terraform output -raw cluster_arn) && \
	SUBNET_ID=$$(cd terraform && terraform output -raw subnet_id) && \
	aws ecs run-task \
		--task-definition $${TASK_DEF_ARN} \
		--cluster $${CLUSTER_ARN} \
		--network-configuration "awsvpcConfiguration={subnets=[$${SUBNET_ID}],assignPublicIp=ENABLED}" \
		--launch-type FARGATE \
		--no-cli-pager

run_unsigned_task:
	TASK_DEF_ARN=$$(cd terraform && terraform output -raw unsigned_task_arn) && \
	CLUSTER_ARN=$$(cd terraform && terraform output -raw cluster_arn) && \
	SUBNET_ID=$$(cd terraform && terraform output -raw subnet_id) && \
	aws ecs run-task \
		--task-definition $${TASK_DEF_ARN} \
		--cluster $${CLUSTER_ARN} \
		--network-configuration "awsvpcConfiguration={subnets=[$${SUBNET_ID}],assignPublicIp=ENABLED}" \
		--launch-type FARGATE \
		--no-cli-pager

task_status:
	CLUSTER_ARN=$$(cd terraform && terraform output -raw cluster_arn) && \
	echo "STOPPED tasks" && \
	aws ecs list-tasks --cluster $$CLUSTER_ARN --desired-status STOPPED && \
	echo "RUNNING tasks" && \
	aws ecs list-tasks --cluster $$CLUSTER_ARN --desired-status RUNNING

################################################################################
# Setup and cleanup
################################################################################

.PHONY: key_gen sign verify_signed verify_unsigned ecr_auth clean stop_tasks

key_gen:
	cosign generate-key-pair --kms awskms:///alias/$(KEY_ALIAS)

sign: ecr_auth
	cosign sign --key awskms:///alias/$(KEY_ALIAS) ${IMAGE_URL_SIGNED}

verify_signed: ecr_auth
	cosign verify --key awskms:///alias/$(KEY_ALIAS) ${IMAGE_URL_SIGNED}

verify_unsigned: ecr_auth
	cosign verify --key awskms:///alias/$(KEY_ALIAS) ${IMAGE_URL_UNSIGNED}

.SILENT: ecr_auth
ecr_auth:
	REGISTRY_URL="$(ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com"; \
	aws ecr get-login-password | \
		docker login --username AWS --password-stdin $$REGISTRY_URL

clean:
	rm -f ./cosign-ecs-function/cosign-ecs-function ${PACKAGED_TEMPLATE}

stop_tasks:
	CLUSTER_ARN=$$(cd terraform && terraform output -raw cluster_arn) && \
	aws ecs list-tasks --cluster $$CLUSTER_ARN --desired-status RUNNING --output text --query taskArns | \
		xargs --no-run-if-empty --max-args 1 \
			aws ecs stop-task --no-cli-pager --cluster $$CLUSTER_ARN --task
