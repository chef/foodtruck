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
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	eventhub "github.com/Azure/azure-event-hubs-go/v3"
	"github.com/spf13/cobra"
)

// listenCmd represents the listen command
var listenCmd = &cobra.Command{
	Use:   "listen",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		offset := "0"

		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}

		fmt.Println("hostname:", hostname)

		connStr := "" + hostname
		hub, err := eventhub.NewHubFromConnectionString(connStr)

		if err != nil {
			fmt.Println(err)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		// send a single message into a random partition
		err = hub.Send(ctx, eventhub.NewEventFromString("hello, world!"))
		if err != nil {
			fmt.Println(err)
			return
		}

		handler := func(c context.Context, event *eventhub.Event) error {
			fmt.Println("Handler!")
			fmt.Println(string(event.Data))
			return nil
		}

		// listen to each partition of the Event Hub
		runtimeInfo, err := hub.GetRuntimeInformation(ctx)
		if err != nil {
			fmt.Println(err)
			return
		}

		for _, partitionID := range runtimeInfo.PartitionIDs {
			// Start receiving messages
			//
			// Receive blocks while attempting to connect to hub, then runs until listenerHandle.Close() is called
			// <- listenerHandle.Done() signals listener has stopped
			// listenerHandle.Err() provides the last error the receiver encountered
			// listenerHandle
			_, err := hub.Receive(ctx, partitionID, handler, eventhub.ReceiveWithStartingOffset(offset))
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println("Receive!")
		}

		// Wait for a signal to quit:
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, os.Interrupt, os.Kill)
		<-signalChan

		err = hub.Close(context.Background())
		if err != nil {
			fmt.Println(err)
		}
	},
}

func init() {
	rootCmd.AddCommand(listenCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// listenCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// listenCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
