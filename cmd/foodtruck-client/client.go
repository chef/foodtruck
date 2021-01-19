package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/chef/foodtruck/pkg/foodtruckhttp"
	"github.com/chef/foodtruck/pkg/models"
	"github.com/chef/foodtruck/pkg/provider"
)

type Config struct {
	Node          models.Node   `json:"node"`
	BaseURL       string        `json:"base_url"`
	ProvidersPath string        `json:"providers_path"`
	Interval      time.Duration `json:"interval"`
}

func main() {
	config := Config{
		BaseURL: "http://localhost:1323",
		Node: models.Node{
			Organization: "neworg",
			Name:         "testnode5",
		},
		Interval: time.Second * 5,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := foodtruckhttp.NewClient(config.BaseURL, config.Node)
	runner := provider.NewExecRunner()
	for {
		select {
		case <-ctx.Done():
			break
		case <-time.After(config.Interval):
			task, err := client.GetNextTask(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[Error]: %s\n", err)
				continue
			}
			fmt.Println("Running task")
			err = client.UpdateNodeTaskStatus(ctx, models.NodeTaskStatus{
				JobID:  task.JobID,
				Status: models.TaskStatusRunning,
			})
			if err != nil {
				fmt.Printf("[Error] %s\n", err)
			}

			if err := runner.Run(ctx, task.Provider, task.Spec); err != nil {
				fmt.Printf("[Error] %s\n", err)
			} else {
				fmt.Println("Task complete")
			}

			err = client.UpdateNodeTaskStatus(ctx, models.NodeTaskStatus{
				JobID:  task.JobID,
				Status: models.TaskStatusComplete,
			})
			if err != nil {
				fmt.Printf("[Error] %s\n", err)
			}

		}
	}
}
