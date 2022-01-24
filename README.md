# cosign-ecs-verify

![](aws-ecs-cosign-verify.png)

1. Start an ECS task in the cluster
2. The task definition has the container image stored in ECR
3. EventBridge sends a notification to Lambda
4. Cluster and Task definition is sent to function 
5. KMS key that has signed an image 
6. Lambda function evaluates if container image is signed w/ KMS
7. If not signed with specified key it does two things
   1. Stop task definition and deregister the service
   2. SNS notification email to alert that the service/task has been stopped