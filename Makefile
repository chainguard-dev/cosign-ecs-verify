NAME ?= cosign-ecs
IMAGE ?= distroless-base
VERSION ?= 0.0.3
AWS_REGION ?= us-west-2
AWS_DEFAULT_REGION ?= us-west-2
ACCOUNT_ID ?= $(shell aws sts get-caller-identity --query Account --output text)
PACKAGED_TEMPLATE = packaged.yml
EVENT ?= event.json
KEY_NAME = cosign-aws

export

.PHONY: aws_account
aws_account:
	${ACCOUNT_ID}

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
	terraform plan -var="name=${NAME}" -var="image_name=${IMAGE}" -var="image_version=${VERSION}"  -out=plan.out

tf_apply:
	cd terraform/ && \
	terraform apply -var="name=${NAME}" -var="image_name=${IMAGE}" -var="image_version=${VERSION}" -auto-approve

tf_fmt:
	cd terraform/ && \
	terraform fmt

tf_destroy:
	cd terraform/ && \
	terraform destroy -var="name=${NAME}" -var="image_name=${IMAGE}" -var="image_version=${VERSION}"  -auto-approve

tf_refresh:
	cd terraform/ && \
	terraform refresh -var="name=${NAME}" -var="image_name=${IMAGE}" -var="image_version=${VERSION}"

go_build:
	cd ./cosign-ecs-function && go mod tidy && \
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o cosign-ecs-function .

clean:
	rm -f ./cosign-ecs-function/cosign-ecs-function ${PACKAGED_TEMPLATE}

sam_build: go_build
	sam build

sam_package: sam_build
	sam package --config-file samconfig.toml --template-file template.yml --output-template-file ${PACKAGED_TEMPLATE} --resolve-s3

sam_deploy: sam_package
	sam deploy --config-file samconfig.toml --parameter-overrides KeyId=${KeyId} --template-file template.yml --stack-name cosign-verify --template-file ${PACKAGED_TEMPLATE} --capabilities CAPABILITY_IAM --resolve-s3

sam_local: sam_build
	sam local invoke -e ${EVENT} --template template.yml

sam_local_debug: sam_build
	sam local invoke -e ${EVENT} --template template.yml --debug

sam_delete:
	sam delete \
		--stack-name ${NAME}-stack \
		--region ${AWS_REGION} \
		--no-prompts
#  if --no-prompts, it ignores $AWS_REGION

run_signed_task:
	aws ecs run-task --task-definition "arn:aws:ecs:us-west-2:$(ACCOUNT_ID):task-definition/cosign-ecs-task-definition:2" --cluster $(NAME)-cluster --network-configuration "awsvpcConfiguration={subnets=[$(SUBNET_ID)],securityGroups=[$(SEC_GROUP_ID)],assignPublicIp=ENABLED}" --launch-type FARGATE

run_unsigned_task:
	aws ecs run-task --task-definition "arn:aws:ecs:us-west-2:$(ACCOUNT_ID):task-definition/cosign-ecs-task-definition:7" --cluster $(NAME)-cluster --network-configuration "awsvpcConfiguration={subnets=[$(SUBNET_ID)],securityGroups=[$(SEC_GROUP_ID)],assignPublicIp=ENABLED}" --launch-type FARGATE

sign: ecr_auth
	cosign sign --key awskms:///alias/$(NAME) $(ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE):$(VERSION)

key_gen:
	cosign generate-key-pair --kms awskms:///alias/$(KEY_NAME) $(ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE):$(VERSION)

verify: key_gen ecr_auth
	cosign verify --key cosign.pub $(ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE):$(VERSION)

.SILENT: ecr_auth
ecr_auth:
	docker login --username AWS -p $(shell aws ecr get-login-password --region $(AWS_REGION) ) $(ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com
