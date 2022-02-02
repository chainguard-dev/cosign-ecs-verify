package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/sns"
	"log"
	"os"
)

type NotificationMessage struct {
	Message           string
	ClusterArn        string
	TaskDefinitionArn string
	TaskArn           string
}

func stopTask(clusterArn string, taskArn string) error {
	// Stop the task
	stopTaskInput := ecs.StopTaskInput{Cluster: aws.String(clusterArn),
		Reason: aws.String("lambda stopping ecs task"), Task: aws.String(taskArn)}

	var svc = ecs.New(session.Must(session.NewSession()))

	_, err := svc.StopTask(&stopTaskInput)
	if err != nil {
		return err
	}
	return nil
}

func sendNotificationEvent(clusterArn, taskDefinitionArn, taskArn string) {
	if message, err := marshalNotificationMessage(clusterArn, taskDefinitionArn, taskArn); err == nil {
		publishInput := sns.PublishInput{
			Message:  aws.String(string(message)),
			Subject:  aws.String("Issues with Container image in Cluster"),
			TopicArn: aws.String(os.Getenv("SNS_TOPIC_ARN")),
		}

		mySession := session.Must(session.NewSession())
		var svc = sns.New(mySession)

		if _, err := svc.Publish(&publishInput); err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				log.Printf("[ERROR] Publishing SNS Message: %v", aerr.Error())
			} else {
				log.Printf("[ERROR] Publishing SNS Message: %v", err.Error())
			}
			return
		}
	}
}

func marshalNotificationMessage(clusterArn, taskDefinitionArn, taskArn string) ([]byte, error) {
	message := fmt.Sprintf("Task %v Attempted to run an unsigned container", taskArn)
	return json.Marshal(NotificationMessage{message, clusterArn, taskDefinitionArn, taskArn})
}

func getKeyID(accountID, region string) (string, error) {
	//Generate the public key from KMS ARN
	keyID := os.Getenv("COSIGN_KEY")
	if len(keyID) == 0 {
		return "", errors.New("KMS ARN is empty")
	}
	log.Printf("[INFO] Key Alias ARN: %v", keyID)
	svc := kms.New(session.Must(session.NewSession()))
	input := &kms.DescribeKeyInput{
		KeyId: aws.String(keyID),
	}

	result, err := svc.DescribeKey(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case kms.ErrCodeNotFoundException:
				log.Printf("[ERROR] AWS Error code %v Error %v", kms.ErrCodeNotFoundException, aerr.Error())
			case kms.ErrCodeInvalidArnException:
				log.Printf("[ERROR] AWS Error code %v Error %v", kms.ErrCodeInvalidArnException, aerr.Error())
			case kms.ErrCodeDependencyTimeoutException:
				log.Printf("[ERROR] AWS Error code %v Error %v", kms.ErrCodeDependencyTimeoutException, aerr.Error())
			case kms.ErrCodeInternalException:
				log.Printf("[ERROR] AWS Error code %v Error %v", kms.ErrCodeInternalException, aerr.Error())
			default:
				log.Printf("[ERROR] AWS Error %v", aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			log.Printf("[ERROR] AWS Error %v", err.Error())
		}
		return "", err
	}

	log.Printf("[INFO] Key look up Key ID %v", *result.KeyMetadata.KeyId)

	return *result.KeyMetadata.KeyId, nil
}
