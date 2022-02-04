# cosign-ecs-verify

A prototype analog of a [Kubernetes admission controller] for Amazon [Elastic
Container Service (ECS)][ECS] which:

- Terminates any running images without a valid [cosign] signature.
- Sends alerts to an [SNS] topic (which can forward to your email).
- Supports configurable keys to use for signature validation.

**NOTE:** This is **Proof of Concept code** and not yet production-ready. In the
event of a bug or misconfiguration, it can prevent *any* ECS tasks from running.
This system intercepts tasks as they start (not before) and may not fully
prevent unsigned containers.

If you'd like to learn more or give feedback, please file an issue or send an
email to [interest@chainguard.dev].

[Kubernetes admission controller]: https://kubernetes.io/docs/reference/access-authn-authz/admission-controllers/
[ECS]: https://aws.amazon.com/ecs/
[cosign]: https://github.com/sigstore/cosign
[KMS]: https://aws.amazon.com/kms/
[SNS]: https://aws.amazon.com/sns/
[interest@chainguard.dev]: mailto:interest@chainguard.dev

## How it works

![](aws-ecs-cosign-verify.png)

This system comprises a Lambda function (4) which listens to EventBridge events
(3) triggered on every ECS task run (1). The function checks that the task's
container image has a valid [cosign] signature in ECR (2) with a specific public
key, provided directly or stored in KMS (5). If the check fails, the function
terminates the task and sends a notification to an SNS topic (7) which you can
subscribe to via email.

In order:

1. Start an ECS task in the cluster
2. The task definition has the container image stored in ECR
3. EventBridge sends a notification to Lambda
4. Cluster and Task definition is sent to function 
5. KMS key that has signed an image 
6. Lambda function evaluates if container image is signed w/ KMS
7. If not signed with specified key it does two things
   1. Stop task definition
   2. SNS notification email to alert that the service/task has been stopped

## Quickstart

### Requirements and preliminaries.

You will need the following tools installed:

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

### Deploy
To deploy, run:

```shell
make sam_deploy
```

It will check against the key in `cosign.pub` (see detailed instructions for how
to change this).

### Test it

To test, we need an ECS cluster on which to run our signed/unsigned tasks.
The `terraform` subdirectory contains a Terraform template for such a cluster,
with corresponding task definitions. First, initialize (to download the AWS
provider for Terraform), then deploy:

``` shell
make tf_init
make tf_apply  # run `make tf_plan` to see the plan first
```

We can then run our tasks (these will run two public "hello world" Alpine images
with and without a signature):

``` shell
make run_unsigned_task
make run_signed_task
```

Check that it worked. You should see the unsigned task in the `STOPPED` tasks
and the signed task in the `RUNNING` tasks:

``` shell
make task_status
```

### Cleanup

To clean up, run:

``` shell
make stop_tasks
make tf_destroy
make sam_delete
```

## Detailed instructions

`cosign-ecs-verify` uses a SAM template (`template.yml`) to create:

- A Lambda function (source in `cosign-ecs-function/`) which:
  - Runs on every ECS task state change, triggered by EventBridge.
  - Gets the key for signature verification.
  - For each container in the event:
    - Verifies the container image, terminating the task and sending a
      notification if it is invalid.
- An Amazon [SNS] topic: if the function stops an unsigned container image, it
  will send a message to this topic.
  - You can [configure email notifications][sns-email] for this topic to be
    alerted whenever an unverified image is stopped.
  - Messages sent to the topic are [encrypted using a key in KMS][sns-kms].
  
To configure, run `sam deploy` with either the `KeyArn` set to a KMS key to use, or
`KeyPem` set to a full public key in PEM format. In the provided `Makefile`, we
hardcode the key in `cosign.pub`.

[SNS]: https://aws.amazon.com/sns/
[sns-email]: https://docs.aws.amazon.com/sns/latest/dg/sns-email-notifications.html
[sns-kms]: https://aws.amazon.com/blogs/compute/encrypting-messages-published-to-amazon-sns-with-aws-kms/

We provide a test ECS cluster configuration (in `terraform/`), containing;

- An ECS cluster:
  - Configured to log to CloudWatch (which will trigger the Lambda via EventBridge).
  - Associated task definitions `signed` and `unsigned`, which (by default) run images.
- KMS keys and IAM permissions for the above.

If you would like to use your own images (for example, from a [previous post on
the Chainguard blog][previous-blog]), export `$IMAGE_URL_SIGNED` and
`$IMAGE_URL_UNSIGNED` before running `make tf_apply`. We give instructions below
for making your own to test.

[previous-blog]: https://blog.chainguard.dev/cosign-image-signing-in-aws-codepipeline/

### Key Pair for `cosign`

We need a key against which to verify image signatures. If you have an existing keypair for cosign in AWS KMS, set it:

``` shell
export KEY_ALIAS=my-key
```

Otherwise, we can make one:

``` shell
export KEY_ALIAS=my-key
export AWS_SDK_LOAD_CONFIG=true
make key_gen
```

### Signed and unsigned images

To see `cosign-ecs-verify` in action, you need an example of a signed and
unsigned image. Here, we'll build two simple images, push them to Amazon [ECR],
and sign only one.

[ECR]: https://aws.amazon.com/ecr/

First, login to ECR with Docker. We recommend using a [credential helper] for
docker, but we also provide a make target `make ecr_auth` that will authenticate
you to the default registry.

[credential helper]: https://aws.amazon.com/blogs/compute/authenticating-amazon-ecr-repositories-for-docker-cli-with-credential-helper/

Then, we can create a repository for the signed/unsigned images.

```shell
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

And sign *only one of them* (make sure you have a key pair):

``` shell
cosign sign --key awskms:///alias/$KEY_ALIAS $IMAGE_URL_SIGNED
```

Now, you can proceed with the Terraform instructions.

### Local Development and Testing

We require Go 1.17 for development.

You can also use the SAM local feature to run in a simulated Lambda environment:

``` shell
make sam_local
make sam_local_debug
```

### License

[Apache License 2.0](LICENSE)
