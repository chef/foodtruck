/*
Copyright © 2020 Chef Software, Inc <success@chef.io>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package azeventhub

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	mgmt "github.com/Azure/azure-sdk-for-go/services/eventhub/mgmt/2017-04-01/eventhub"
	"github.com/Azure/go-autorest/autorest/azure"

	azauth "github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/Azure/azure-amqp-common-go/v3/aad"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
)

const (
	Location = "eastus2"
)

type azureEventHub struct {
	client                  *eventhub.Hub
	mgmtClient              *mgmt.EventHubsClient
	partitionIDs            []string
	partitionCount          int64
	messageRetentionInDays  int64
	hubname                 string
	env                     map[string]string
	resourceManagerEndpoint string
}

// NewAzureEventHub creates a new Azure Event Hub Object.
func NewAzureEventHub(partitionCount, messageRetentionInDays int64) *azureEventHub {
	fmt.Println("Creating new Azure Event Hubs Client...")
	az := new(azureEventHub)
	az.env = make(map[string]string)
	az.resourceManagerEndpoint = azure.PublicCloud.ResourceManagerEndpoint
	az.partitionCount = partitionCount
	az.messageRetentionInDays = messageRetentionInDays

	az.getEnvVars()
	az.mgmtClient = az.getEventHubMgmtClient()

	return az
}

func (az *azureEventHub) getEnvVars() {
	fmt.Println("Validating Environment Variables...")
	reqEnvVars := []string{
		"AZURE_TENANT_ID",
		"AZURE_CLIENT_ID",
		"AZURE_CLIENT_SECRET",
		"AZURE_SUBSCRIPTION_ID",
		"AZURE_EVENTHUB_LOCATION",
		"AZURE_EVENTHUB_RESOURCEGROUP",
		"AZURE_EVENTHUB_NAMESPACE",
	}
	missingEnvVars := make([]string, 0, 10)

	for _, key := range reqEnvVars {
		v := os.Getenv(key)
		if v == "" {
			// fmt.Println("Missing: " + key)
			missingEnvVars = append(missingEnvVars, key)
		} else {
			// fmt.Println("Found: " + key)
			az.env[key] = v
		}
	}
	// fmt.Printf("Length: %d\n", len(missingEnvVars))
	if len(missingEnvVars) > 0 {
		panic("Required Environment Variables are missing: " + strings.Join(missingEnvVars, ", "))
	}
}

func (az *azureEventHub) getEventHubMgmtClient() *mgmt.EventHubsClient {
	// fmt.Println("Creating new Azure Event Hubs Management Client in: ")
	// fmt.Println("  Subscription ID: " + az.env["AZURE_SUBSCRIPTION_ID"])
	// fmt.Println("  Endpoint: ", az.resourceManagerEndpoint)
	mgmtClient := mgmt.NewEventHubsClientWithBaseURI(az.resourceManagerEndpoint, az.env["AZURE_SUBSCRIPTION_ID"])
	// fmt.Println("BaseURI: " + mgmtClient.BaseURI)
	a, err := azauth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	mgmtClient.Authorizer = a
	return &mgmtClient
}

func (az *azureEventHub) Register(hubname string) error {
	az.hubname = hubname
	fmt.Println("Registering hub with name " + az.hubname)

	hubModel, err := az.mgmtClient.Get(context.Background(), az.env["AZURE_EVENTHUB_RESOURCEGROUP"], az.env["AZURE_EVENTHUB_NAMESPACE"], az.hubname)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		newHub := &mgmt.Model{
			Name: &az.hubname,
			Properties: &mgmt.Properties{
				PartitionCount:         &az.partitionCount,
				MessageRetentionInDays: &az.messageRetentionInDays,
			},
		}
		// fmt.Printf("Model::Name %v\n", *newHub.Name)
		// fmt.Printf("Model::ParitionCount %v\n", *newHub.Properties.PartitionCount)
		// fmt.Printf("Model::MessageRetentionInDays %v\n", *newHub.Properties.MessageRetentionInDays)

		hubModel, err = az.mgmtClient.CreateOrUpdate(context.Background(), az.env["AZURE_EVENTHUB_RESOURCEGROUP"], az.env["AZURE_EVENTHUB_NAMESPACE"], az.hubname, *newHub)
		if err != nil {
			return err
		}
	}

	// fmt.Println("Set PartitionIDs")
	// fmt.Printf("Partition IDs: %v", *hubModel.PartitionIds)
	az.partitionIDs = *hubModel.PartitionIds

	// fmt.Println("Get Provider")
	provider, err := aad.NewJWTProvider(aad.JWTProviderWithEnvironmentVars())
	if err != nil {
		return err
	}
	// fmt.Println("Get Hub Client")
	hub, err := eventhub.NewHub(az.env["AZURE_EVENTHUB_NAMESPACE"], hubname, provider)
	if err != nil {
		return err
	}
	az.client = hub
	return nil
}

func (az *azureEventHub) StartListener() (chan string, error) {
	fmt.Println("Starting Listener...")
	events := make(chan string)

	handler := func(ctx context.Context, event *eventhub.Event) error {
		events <- string(event.Data)
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, partitionID := range az.partitionIDs {
		_, err := az.client.Receive(ctx, partitionID, handler, eventhub.ReceiveWithLatestOffset())
		if err != nil {
			return nil, err
		}
	}
	cancel()

	fmt.Println("Azure Event Hub Listener Started :: Listening to: " + az.hubname)
	return events, nil
}

func (az *azureEventHub) StopListener() error {
	return nil
}

func (az *azureEventHub) Deregister() error {
	return nil
}

func (az *azureEventHub) SendOrder(order string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := az.client.Send(ctx, eventhub.NewEventFromString(order))

	return err
}
