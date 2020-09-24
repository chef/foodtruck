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
	"os"
	"context"
	"time"
	"log"
	"fmt"

	mgmt "github.com/Azure/azure-sdk-for-go/services/eventhub/mgmt/2017-04-01/eventhub"
	"github.com/Azure/go-autorest/autorest/azure"

	azauth "github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/Azure/azure-amqp-common-go/v3/aad"

	"github.com/Azure/azure-event-hubs-go/v3"
)

const (
	Location          = "eastus2"
	ResourceGroupName = "foodtruck-poc"
)

func RegisterNode() {
	hubname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, err = ensureEventHub(ctx, hubname)
	if err != nil {
		panic(err)
	}

}

func ListenToHub() {
	hub, partitions := initHub()
	exit := make(chan struct{})

	handler := func(ctx context.Context, event *eventhub.Event) error {
		text := string(event.Data)
		if text == "exit\n" {
			fmt.Println("Oh snap!! Someone told me to exit!")
			exit <- *new(struct{})
		} else {
			fmt.Println(string(event.Data))
		}
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for _, partitionID := range partitions {
		_, err := hub.Receive(ctx, partitionID, handler, eventhub.ReceiveWithLatestOffset())
		if err != nil {
			fmt.Println("Error: ", err)
			return
		}
	}
	cancel()

	fmt.Println("I am listening...")

	select {
	case <-exit:
		fmt.Println("closing after 2 seconds")
		select {
		case <-time.After(2 * time.Second):
			return
		}
	}

}

func mustGetenv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("Environment variable '" + key + "' required for integration tests.")
	}
	return v
}

func getEventHubMgmtClient() *mgmt.EventHubsClient {
	subID := mustGetenv("AZURE_SUBSCRIPTION_ID")
	client := mgmt.NewEventHubsClientWithBaseURI(azure.PublicCloud.ResourceManagerEndpoint, subID)
	a, err := azauth.NewAuthorizerFromEnvironment()
	if err != nil {
		log.Fatal(err)
	}
	client.Authorizer = a
	return &client
}

func ensureEventHub(ctx context.Context, name string) (*mgmt.Model, error) {
	namespace := mustGetenv("EVENTHUB_NAMESPACE")
	client := getEventHubMgmtClient()
	hub, err := client.Get(ctx, ResourceGroupName, namespace, name)

	partitionCount := int64(2)
	messageRetentionInDays := int64(1)
	if err != nil {
		newHub := &mgmt.Model{
			Name: &name,
			Properties: &mgmt.Properties{
				PartitionCount: &partitionCount,
				MessageRetentionInDays: &messageRetentionInDays,
			},
		}

		hub, err = client.CreateOrUpdate(ctx, ResourceGroupName, namespace, name, *newHub)
		if err != nil {
			return nil, err
		}
	}
	return &hub, nil
}

func initHub() (*eventhub.Hub, []string) {
	namespace := mustGetenv("EVENTHUB_NAMESPACE")
	hubname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	hubMgmt, err := ensureEventHub(context.Background(), hubname)
	if err != nil {
		log.Fatal(err)
	}

	provider, err := aad.NewJWTProvider(aad.JWTProviderWithEnvironmentVars())
	if err != nil {
		log.Fatal(err)
	}
	hub, err := eventhub.NewHub(namespace, hubname, provider)
	if err != nil {
		panic(err)
	}
	return hub, *hubMgmt.PartitionIds
}

// provider, err := sas.NewTokenProvider(sas.TokenProviderWithEnvironmentVars())
// if err != nil {
// 	fmt.Println(err)
// 	return
// }
// ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
// hubMgmt, err := ensureEventHub(context.Background(), hostname)
// if err != nil {
// 	log.Fatal(err)
// }

// hub, err := eventhubs.NewHub("foodtruck", hostname, provider)

// defer hub.Close(ctx)
// defer cancel()
// if err != nil {
// 	log.Fatalf("failed to get hub %s\n", err)
// }

// ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
// defer cancel()

// // send a single message into a random partition
// err = hub.Send(ctx, eventhub.NewEventFromString("hello, world!"))
// if err != nil {
// 	fmt.Println(err)
// 	return
// }

// handler := func(c context.Context, event *eventhub.Event) error {
// 	fmt.Println("Handler!")
// 	fmt.Println(string(event.Data))
// 	return nil
// }

// // listen to each partition of the Event Hub
// runtimeInfo, err := hub.GetRuntimeInformation(ctx)
// if err != nil {
// 	fmt.Println(err)
// 	return
// }

// for _, partitionID := range runtimeInfo.PartitionIDs {
// 	// Start receiving messages
// 	//
// 	// Receive blocks while attempting to connect to hub, then runs until listenerHandle.Close() is called
// 	// <- listenerHandle.Done() signals listener has stopped
// 	// listenerHandle.Err() provides the last error the receiver encountered
// 	// listenerHandle
// 	_, err := hub.Receive(ctx, partitionID, handler, eventhub.ReceiveWithStartingOffset(offset))
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	fmt.Println("Receive!")
// }

// // Wait for a signal to quit:
// signalChan := make(chan os.Signal, 1)
// signal.Notify(signalChan, os.Interrupt, os.Kill)
// <-signalChan

// err = hub.Close(context.Background())
// if err != nil {
// 	fmt.Println(err)
// }