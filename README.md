# cosign-ecs-verify

In this demo, we build an analog of a [Kubernetes admission controller] for
Amazon [Elastic Container Service (ECS)][ECS] that checks all images to be run
for a valid [cosign] signature with a given key in AWS [KMS].

**NOTE:** This is **Proof of Concept code** and as such shouldn't be used in
production. In the event of misconfiguration or a bug, it can prevent all ECS
containers from running. Please let us know if you have any feedback or interest in learning more about this [interest@chainguard.dev](mailto:interest@chainguard.dev)

[Kubernetes admission controller]: https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/
[ECS]: https://aws.amazon.com/ecs/
[cosign]: https://github.com/sigstore/cosign
[KMS]: https://aws.amazon.com/kms/

## How it works

![](aws-ecs-cosign-verify.png)

1. Start an ECS task in the cluster
2. The task definition has the container image stored in ECR
3. EventBridge sends a notification to Lambda
4. Cluster and Task definition is sent to function 
5. KMS key that has signed an image 
6. Lambda function evaluates if container image is signed w/ KMS
7. If not signed with specified key it does two things
   1. Stop task definition
   2. SNS notification email to alert that the service/task has been stopped

## Requirements and preliminaries.

For this demo, you will need the following tools installed:

- `make` (e.g., [GNU make])
- [Terraform]: for local testing
- [AWS CLI] and [AWS SAM CLI]: for deploying
- [`cosign`]: for generating keys
- [`docker`]: if you need to make images

[AWS CLI]: https://aws.amazon.com/cli/
[AWS SAM CLI]: https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/serverless-sam-cli-install.html
[GNU make]: https://www.gnu.org/software/make/
[Terraform]: https://www.terraform.io/downloads
[`cosign`]: https://github.com/sigstore/cosign
[`docker`]: https://docs.docker.com/get-docker/

You should [configure the AWS CLI] for your project and account.

[configure the AWS CLI]: https://docs.aws.amazon.com/cli/latest/reference/configure/

We need a key against which to verify image signatures. If you have an existing
keypair for cosign in AWS KMS, set it:

``` shell
export KEY_ALIAS=my-key
```

Otherwise, we can make one:

``` shell
export KEY_ALIAS=my-key
export AWS_SDK_LOAD_CONFIG=true
make key_gen
```

## Deploy

To deploy, run:

```shell
make sam_deploy
```

This uses a SAM template (`template.yml`) to create:

- The Serverless function (source in cosign-ecs-function)
  - Triggered on cloud watch event: ecs task/container state change
  - for each container in the event
    - get the key corresponding to the region
    - verify the container image
- An Amazon [SNS] topic: if the function stops an unsigned container image, it
  will send a message to this topic.
  - You can [configure email notifications][sns-email] for this topic to be
    alerted whenever an unverified image is stopped.
  - Messages sent to the topic are [encrypted using a key in KMS][sns-kms].

[SNS]: https://aws.amazon.com/sns/
[sns-email]: https://docs.aws.amazon.com/sns/latest/dg/sns-email-notifications.html
[sns-kms]: https://aws.amazon.com/blogs/compute/encrypting-messages-published-to-amazon-sns-with-aws-kms/
    

## Test it

### Signed and unsigned images

To see this demo in action, you need an example of a signed and unsigned image.

If you already have a such images (ex. from a [previous post on the Chainguard blog][previous-blog]), we can use those:

[previous-blog]: https://blog.chainguard.dev/cosign-image-signing-in-aws-codepipeline/

```shell
export IMAGE_URL_SIGNED=...
export IMAGE_URL_UNSIGNED=...
```

Otherwise, we'll build two simple images, push them to Amazon [ECR], and sign only one.

[ECR]: https://aws.amazon.com/ecr/

First, login to ECR with Docker. We recommend using a [credential helper] for
docker, but we also provide a Make target `make ecr_auth` that will authenticate

``` shell
aws ecr create-repository --repository-name $REPO_NAME
```
you to the default registry.

[credential helper]: https://aws.amazon.com/blogs/compute/authenticating-amazon-ecr-repositories-for-docker-cli-with-credential-helper/

Then, we can create a repository for the signed/unsigned images.

```shell
REPO_NAME=ecr-demo-image
REPO_URL=$(aws ecr create-repository \
    --repository-name $REPO_NAME \
    --query repository.repositoryUri \
    --output text)
```
Finally, we can build and push two simple images (see `Dockerfile`):

``` shell
# Export these so we can make ECR task definitions for running them.
export IMAGE_URL_SIGNED=$REPO_URL:signed
export IMAGE_URL_UNSIGNED=$REPO_URL:unsigned
# Make 2 example images and push both.
# The --build-arg is to make sure the images have different digests.
docker build . --build-arg signed=true --tag $IMAGE_URL_SIGNED
docker build . --build-arg signed=false --tag $IMAGE_URL_UNSIGNED
docker push $IMAGE_URL_SIGNED
docker push $IMAGE_URL_UNSIGNED
```

And sign *only one of them*:

``` shell
export AWS_SDK_LOAD_CONFIG=true
cosign sign --key awskms:///alias/$KEY_ALIAS $IMAGE_URL_SIGNED
```

### Deploy a cluster and run tasks

The `terraform` subdirectory contains a Terraform template for an ECS cluster
and task definitions for running our signed/unsigned tasks. First, initialize
it (this will download required providers):

``` shell
make tf_init
```

Then, deploy the template:

``` shell
make tf_apply  # run `make tf_plan` to see the plan first
```

We can then run our tasks:

``` shell
make run_unsigned_task
make run_signed_task
```

*Note:* this will run on the tasks on a subnet of the [default VPC].

[default VPC]: https://docs.aws.amazon.com/vpc/latest/userguide/default-vpc.html

Check:

``` shell
make task_status
```

You should see the unsigned task in the `STOPPED` tasks and the signed task in the `RUNNING` tasks.


### Cleanup

``` shell
make stop_tasks
make tf_destroy
make sam_delete
# If you created an ECR repository (--force deletes the images in it):
aws ecr delete-repository --repository-name $REPO_NAME --force
# Clean up Docker mages locally
docker images "*/$REPO_NAME" | xargs docker image rm --force
# To clean up the KMS key used for signing:
KEY_ID=$(aws kms describe-key --alias $KEY_ALIAS)
aws kms delete-alias $KEY_ALIAS
aws kms disable-key $KEY_ID
```


## Local Dev

- Go 1.17

``` shell
make sam_local 
make sam_local_debug
```

License

[Apache License 2.0](LICENSE)