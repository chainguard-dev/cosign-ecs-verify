package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sns"
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
