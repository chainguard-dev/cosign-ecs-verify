package main

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"
	"log"
)

func handler(event events.CloudWatchEvent) {

	var eventBridgeECSTaskStatusChangeEventDetail EventBridgeECSTaskStatusChangeEventDetail
	err := json.Unmarshal(event.Detail, &eventBridgeECSTaskStatusChangeEventDetail)
	if err != nil {
		log.Fatal("error during event unmarshalling:", err)
	}

	clusterArn := eventBridgeECSTaskStatusChangeEventDetail.ClusterArn
	taskArn := eventBridgeECSTaskStatusChangeEventDetail.TaskArn
	taskDefinitionArn := eventBridgeECSTaskStatusChangeEventDetail.TaskDefinitionArn

	var accountId = getAccountId()

	log.Printf("Cluster: %v\n", clusterArn)
	log.Printf("taskArn: %v\n", taskArn)
	log.Printf("taskDefinitionArn: %v\n", taskDefinitionArn)
	log.Printf("accountId: %v\n", accountId)
}

func main() {
	lambda.Start(handler)
}

func getAccountId() string {
	svc := sts.New(session.New())
	input := &sts.GetCallerIdentityInput{}

	result, err := svc.GetCallerIdentity(input)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			log.Println(awsErr.Error())
		} else {
			log.Println(err.Error())
		}
		log.Fatal("Error getting account id")
	}

	return aws.StringValue(result.Account)
}

type EventBridgeECSTaskStatusChangeEventDetail struct {
	ClusterArn        string `json:"clusterArn"`
	TaskArn           string `json:"taskArn"`
	TaskDefinitionArn string `json:"taskDefinitionArn"`
}
