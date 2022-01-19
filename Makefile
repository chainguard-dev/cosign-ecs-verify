NAME ?=cosign-ecs
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
	terraform plan -var="name=${NAME}" -out=plan.out

tf_apply:
	cd terraform/ && \
	terraform apply -var="name=${NAME}" -auto-approve

tf_destroy:
	cd terraform/ && \
	terraform destroy -var="name=${NAME}" -auto-approve

sign:
	cosign sign --key awskms:///alias/$(NAME) $(ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE):$(VERSION)

key_gen:
	cosign generate-key-pair --kms awskms:///alias/$(NAME) $(ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE):$(VERSION)

verify: key_gen
	cosign verify --key cosign.pub $(ACCOUNT_ID).dkr.ecr.$(AWS_REGION).amazonaws.com/$(IMAGE):$(VERSION)

build:
	sam build

deploy:
	sam deploy

local:
	sam local invoke -e event.json

