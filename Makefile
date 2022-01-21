NAME ?= cosign-ecs
IMAGE ?= distroless-base
VERSION ?= 0.0.3
GOLANG_VERSION ?= 1.17.2
AWS_REGION ?= us-west-2
AWS_DEFAULT_REGION ?= us-west-2
REPO_INFO ?= $(shell git config --get remote.origin.url)
COMMIT_SHA ?= git-(shell git rev-parse --short HEAD)
COSIGN_ROLE_NAME ?= "${NAME}-codebuild"
ACCOUNT_ID ?= $(shell aws sts get-caller-identity --query Account --output text)
PACKAGED_TEMPLATE = packaged.yml
EVENT ?= event.json

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

go_build: ./cosign-ecs-function
	go build -o cosign-ecs-function ./cosign-ecs-function

clean:
	rm -f ./cosign-ecs-function ${PACKAGED_TEMPLATE}

lambda:
	GOOS=linux GOARCH=amd64 ${MAKE} go_build

sam_init:
	aws s3 mb s3://chainguard-${NAME}

sam_build: lambda
	sam build

sam_package: sam_build
	sam package --template-file template.yml --s3-bucket chainguard-${NAME} --output-template-file ${PACKAGED_TEMPLATE}

sam_deploy: sam_package
	sam deploy --template-file template.yml --stack-name cosign-verify --template-file ${PACKAGED_TEMPLATE} --capabilities CAPABILITY_IAM --s3-bucket chainguard-${NAME}

sam_local: sam_build
	sam local invoke -e ${EVENT}

start_task:
	aws ecs start-task --cluster ${NAME}-cluster --task-definition service
