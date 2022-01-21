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
	"strings"
)

func listServices(clusterArn string) (*ecs.ListServicesOutput, error) {
	// Create client
	mySession := session.Must(session.NewSession())
	var svc = ecs.New(mySession)

	listServicesInput := ecs.ListServicesInput{Cluster: aws.String(clusterArn)}
	return svc.ListServices(&listServicesInput)
}

func setServiceDesiredCountToZero(clusterArn string, serviceName string) {
	fmt.Printf("Service %s to be updated with desired capacity set to 0\n", serviceName)

	// Set the desired count to 0
	updateServiceInput := ecs.UpdateServiceInput{Cluster: aws.String(clusterArn),
		Service:      aws.String(serviceName),
		DesiredCount: aws.Int64(0)}

	mySession := session.Must(session.NewSession())
	var svc = ecs.New(mySession)

	_, err := svc.UpdateService(&updateServiceInput)
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
			case ecs.ErrCodeServiceNotFoundException:
				fmt.Println(ecs.ErrCodeServiceNotFoundException, aerr.Error())
			case ecs.ErrCodeServiceNotActiveException:
				fmt.Println(ecs.ErrCodeServiceNotActiveException, aerr.Error())
			case ecs.ErrCodePlatformUnknownException:
				fmt.Println(ecs.ErrCodePlatformUnknownException, aerr.Error())
			case ecs.ErrCodePlatformTaskDefinitionIncompatibilityException:
				fmt.Println(ecs.ErrCodePlatformTaskDefinitionIncompatibilityException, aerr.Error())
			case ecs.ErrCodeAccessDeniedException:
				fmt.Println(ecs.ErrCodeAccessDeniedException, aerr.Error())
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

func stopTask(clusterArn string, taskArn string) {
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
		return
	}
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

func sendNotificationEvent(clusterArn, serviceArn, taskDefinitionArn, taskArn string) {
	if message, err := marshalNotificationMessage(clusterArn, serviceArn, taskDefinitionArn, taskArn); err == nil {
		publishInput := sns.PublishInput{Message: aws.String(string(message)),
			TopicArn: aws.String(os.Getenv("SNS_TOPIC_ARN"))}
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
	ClusterArn        string
	ServiceArn        string
	TaskDefinitionArn string
	TaskArn           string
}

func marshalNotificationMessage(clusterArn, serviceArn, taskDefinitionArn, taskArn string) ([]byte, error) {
	m := NotificationMessage{clusterArn, serviceArn, taskDefinitionArn, taskArn}

	return json.Marshal(m)
}

func parseServiceName(serviceName string) string {
	return strings.Split(serviceName, "/")[2]
}

func listServicesErrorHandler(listServicesError error) {

	if listServicesError != nil {
		if aerr, ok := listServicesError.(awserr.Error); ok {
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
			fmt.Println(listServicesError.Error())
		}
		return
	}
}
