package main

import "time"

type LambdaEvent struct {
	Version    string    `json:"version"`
	ID         string    `json:"id"`
	DetailType string    `json:"detail-type"`
	Source     string    `json:"source"`
	Account    string    `json:"account"`
	Time       time.Time `json:"time"`
	Region     string    `json:"region"`
	Resources  []string  `json:"resources"`
	Detail     Detail    `json:"detail"`
}
type Details struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type Attachments struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Status  string    `json:"status"`
	Details []Details `json:"details"`
}
type Attributes struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type Containers struct {
	ContainerArn      string        `json:"containerArn"`
	LastStatus        string        `json:"lastStatus"`
	Name              string        `json:"name"`
	Image             string        `json:"image"`
	TaskArn           string        `json:"taskArn"`
	NetworkInterfaces []interface{} `json:"networkInterfaces"`
	CPU               string        `json:"cpu"`
	Memory            string        `json:"memory"`
}
type EphemeralStorage struct {
	SizeInGiB int `json:"sizeInGiB"`
}
type ContainerOverrides struct {
	Name string `json:"name"`
}
type Overrides struct {
	ContainerOverrides []ContainerOverrides `json:"containerOverrides"`
}
type Detail struct {
	Attachments          []Attachments    `json:"attachments"`
	Attributes           []Attributes     `json:"attributes"`
	AvailabilityZone     string           `json:"availabilityZone"`
	ClusterArn           string           `json:"clusterArn"`
	Containers           []Containers     `json:"containers"`
	CPU                  string           `json:"cpu"`
	CreatedAt            time.Time        `json:"createdAt"`
	DesiredStatus        string           `json:"desiredStatus"`
	EnableExecuteCommand bool             `json:"enableExecuteCommand"`
	EphemeralStorage     EphemeralStorage `json:"ephemeralStorage"`
	Group                string           `json:"group"`
	LaunchType           string           `json:"launchType"`
	LastStatus           string           `json:"lastStatus"`
	Memory               string           `json:"memory"`
	Overrides            Overrides        `json:"overrides"`
	PlatformVersion      string           `json:"platformVersion"`
	TaskArn              string           `json:"taskArn"`
	TaskDefinitionArn    string           `json:"taskDefinitionArn"`
	UpdatedAt            time.Time        `json:"updatedAt"`
	Version              int              `json:"version"`
}
