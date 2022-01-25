package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/sns"
	"os"
)

func stopTask(clusterArn string, taskArn string) error {
	// Stop the task
	stopTaskInput := ecs.StopTaskInput{Cluster: aws.String(clusterArn),
		Reason: aws.String("lambda stopping ecs task"), Task: aws.String(taskArn)}

	// Create client
	mySession := session.Must(session.NewSession())
	var svc = ecs.New(mySession)

	_, err := svc.StopTask(&stopTaskInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				fmt.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				fmt.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				fmt.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			case ecs.ErrCodeClusterNotFoundException:
				fmt.Println(ecs.ErrCodeClusterNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return err
	}
	return nil
}

func deregisterTaskDefinition(taskDefinitionArn string) {
	deregisterTaskDefinitionInput := ecs.DeregisterTaskDefinitionInput{TaskDefinition: aws.String(taskDefinitionArn)}

	// Create client
	mySession := session.Must(session.NewSession())
	var svc = ecs.New(mySession)

	_, err := svc.DeregisterTaskDefinition(&deregisterTaskDefinitionInput)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ecs.ErrCodeServerException:
				fmt.Println(ecs.ErrCodeServerException, aerr.Error())
			case ecs.ErrCodeClientException:
				fmt.Println(ecs.ErrCodeClientException, aerr.Error())
			case ecs.ErrCodeInvalidParameterException:
				fmt.Println(ecs.ErrCodeInvalidParameterException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return
	}

}

func sendNotificationEvent(clusterArn, taskDefinitionArn, taskArn string) {
	if message, err := marshalNotificationMessage(clusterArn, taskDefinitionArn, taskArn); err == nil {
		publishInput := sns.PublishInput{
			Message:  aws.String(string(message)),
			Subject:  aws.String("Issues with Container image in Cluster"),
			TopicArn: aws.String(os.Getenv("SNS_TOPIC_ARN")),
		}
		// Create client
		mySession := session.Must(session.NewSession())
		var svc = sns.New(mySession)

		if _, err := svc.Publish(&publishInput); err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				fmt.Println(aerr.Error())
			} else {
				fmt.Println(err.Error())
			}
			return
		}
	}
}

type NotificationMessage struct {
	Message           string
	ClusterArn        string
	TaskDefinitionArn string
	TaskArn           string
}

func marshalNotificationMessage(clusterArn, taskDefinitionArn, taskArn string) ([]byte, error) {
	message := fmt.Sprintf("Task Definition Attempted to run an unsigned container")
	m := NotificationMessage{message, clusterArn, taskDefinitionArn, taskArn}

	return json.Marshal(m)
}
