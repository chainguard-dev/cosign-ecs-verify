NAME ?= cosign-ecs-demo
IMAGE ?= distroless-base
KEY_ALIAS ?= ${NAME}-key
VERSION ?= 0.0.3
AWS_REGION ?= us-west-2
ACCOUNT_ID ?= $(shell aws sts get-caller-identity --query Account --output text)
EVENT ?= event.json
IMAGE ?= distroless-base  # TODO: should be able to change this
IMAGE_URL_SIGNED ?= ${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${IMAGE}:0.0.3
IMAGE_URL_UNSIGNED ?= ${ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com/${IMAGE}:unsigned

AWS_DEFAULT_REGION = ${AWS_REGION}
STACK_NAME = ${NAME}-stack
CLUSTER_NAME = ${NAME}-cluster
SAM_TEMPLATE = template.yml
PACKAGED_TEMPLATE = packaged.yml

export

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

go_build:
	cd ./cosign-ecs-function && go mod tidy && \
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o cosign-ecs-function .

clean:
	rm -f ./cosign-ecs-function/cosign-ecs-function ${PACKAGED_TEMPLATE}

sam_build: go_build
	sam build

sam_package: sam_build
	sam package \
		--template-file ${SAM_TEMPLATE} \
		--output-template-file ${PACKAGED_TEMPLATE} \
		--resolve-s3

sam_deploy: sam_package
	sam deploy \
		--template-file ${SAM_TEMPLATE} \
		--resolve-s3 \
		--parameter-overrides KeyAlias=${KEY_ALIAS} \
		--capabilities CAPABILITY_IAM \
		--stack-name ${STACK_NAME}

sam_local: sam_build
	sam local invoke \
		--event ${EVENT} \
		--template ${SAM_TEMPLATE}

sam_local_debug: sam_build
	sam local invoke \
		--event ${EVENT} \
		--template ${SAM_TEMPLATE} \
		--debug

sam_delete:
	sam delete \
		--stack-name ${STACK_NAME} \
		--region ${AWS_REGION} \
		--no-prompts
#  if --no-prompts, it ignores $AWS_REGION


SUBNET_JQ_QUERY = '.values.root_module.resources | map(select(.type == "aws_subnet" and .name == "public")) | .[0].values.id'

run_signed_task:
	SUBNET_ID=$$(cd terraform && terraform show -json | jq $(SUBNET_JQ_QUERY)); \
	aws ecs run-task \
		--task-definition "arn:aws:ecs:us-west-2:$(ACCOUNT_ID):task-definition/cosign-ecs-task-definition:2" \
		--cluster ${CLUSTER_NAME} \
		--network-configuration "awsvpcConfiguration={subnets=[$${SUBNET_ID}],assignPublicIp=ENABLED}" \
		--launch-type FARGATE

run_unsigned_task:
	SUBNET_ID=$$(cd terraform && terraform show -json | jq $(SUBNET_JQ_QUERY)); \
	aws ecs run-task \
		--task-definition "arn:aws:ecs:us-west-2:$(ACCOUNT_ID):task-definition/cosign-ecs-task-definition:7" \
		--cluster $(NAME)-cluster \
		--network-configuration "awsvpcConfiguration={subnets=[$${SUBNET_ID}],assignPublicIp=ENABLED}" \
		--launch-type FARGATE

sign: ecr_auth
	cosign sign --key awskms:///alias/$(KEY_ALIAS) ${IMAGE_URL_SIGNED}

key_gen:
	cosign generate-key-pair --kms awskms:///alias/$(KEY_ALIAS)

verify: key_gen ecr_auth
	cosign verify --key awskms:///alias/$(KEY_ALIAS) ${IMAGE_URL_SIGNED}

.SILENT: ecr_auth
ecr_auth:
	docker login \
		--username AWS \
		--password $(shell aws ecr get-login-password --region $(AWS_REGION)) \  # TODO: password-stdin
		$(ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com
