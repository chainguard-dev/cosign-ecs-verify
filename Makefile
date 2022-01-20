NAME ?= cosign-ecs
IMAGE ?= distroless-base
VERSION ?= 0.0.1
GOLANG_VERSION ?= 1.17.2
AWS_REGION ?= us-west-2
AWS_DEFAULT_REGION ?= us-west-2
REPO_INFO ?= $(shell git config --get remote.origin.url)
COMMIT_SHA ?= git-$(shell git rev-parse --short HEAD)
COSIGN_ROLE_NAME ?= "$(NAME)-codebuild"
ACCOUNT_ID ?= $(shell aws sts get-caller-identity --query Account --output text)

export

.PHONY: aws_account
aws_account:
	$(ACCOUNT_ID)

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

sign:
	cosign sign --key awskms:///alias/$(NAME) $(ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE):$(VERSION)

key_gen:
	cosign generate-key-pair --kms awskms:///alias/$(NAME) $(ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE):$(VERSION)

verify: key_gen
	cosign verify --key cosign.pub $(ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE):$(VERSION)

sam_init:
	aws s3 mb s3://chainguard-${NAME}

sam_build:
	sam build

sam_deploy: sam_build
	sam deploy --template-file template.yml --stack-name cosign-verify --capabilities CAPABILITY_IAM --s3-bucket chainguard-${NAME}

sam_local: sam_build
	sam local invoke -e event.json
