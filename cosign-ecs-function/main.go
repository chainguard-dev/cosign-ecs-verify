package main

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"log"
)

func handler(event events.CloudWatchEvent) {

	var eventDetail Detail
	err := json.Unmarshal(event.Detail, &eventDetail)
	if err != nil {
		log.Fatalf("[ERROR] %v error during event unmarshalling: %v", event.ID, err)
	}

	lambdaEvent := LambdaEvent{
		Version:    event.Version,
		ID:         event.ID,
		DetailType: event.DetailType,
		Source:     event.Source,
		Account:    event.AccountID,
		Time:       event.Time,
		Region:     event.Region,
		Resources:  event.Resources,
		Detail:     eventDetail,
	}

	log.Printf("[INFO] Cluster: %v\n", lambdaEvent.Detail.ClusterArn)
	log.Printf("[INFO] taskArn: %v\n", lambdaEvent.Detail.TaskArn)
	log.Printf("[INFO] taskDefinitionArn: %v\n", lambdaEvent.Detail.TaskDefinitionArn)
	log.Printf("[INFO] accountId: %v\n", lambdaEvent.Account)

	for i := 0; i < len(lambdaEvent.Detail.Containers); i++ {
		log.Printf("[INFO] Container Image %v : %v", i, lambdaEvent.Detail.Containers[i].Image)
		verified, err := Verify(lambdaEvent.Detail.Containers[i].Image)
		if err != nil {
			log.Printf("[ERROR] Verifing image: %v %v", verified, err)
		}
		if verified {
			log.Println("[INFO] VERIFIED")
		} else {
			log.Println("[INFO] NOT VERIFIED")
			//Stop Tasks etc
			err := stopTask(lambdaEvent.Detail.ClusterArn, lambdaEvent.Detail.TaskArn)
			if err != nil {
				log.Printf("[ERROR] Stopping Task %v : %v", lambdaEvent.Detail.TaskArn, err)
			}
			sendNotificationEvent(lambdaEvent.Detail.ClusterArn, lambdaEvent.Detail.TaskDefinitionArn, lambdaEvent.Detail.TaskArn)
		}
	}
}

func main() {
	lambda.Start(handler)
}
