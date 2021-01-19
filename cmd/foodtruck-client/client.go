package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
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

			taskStatus := models.NodeTaskStatus{
				JobID:  task.JobID,
				Result: &models.NodeTaskStatusResult{},
			}
			if err := runner.Run(ctx, task.Provider, task.Spec); err != nil {
				fmt.Printf("[Error] %s\n", err)
				taskStatus.Status = models.TaskStatusFailed
				exitErr := &exec.ExitError{}
				if errors.As(err, &exitErr) {
					taskStatus.Result.Reason = "exit error"
					taskStatus.Result.ExitCode = -1
					if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
						taskStatus.Result.ExitCode = status.ExitStatus()
					}
				} else {
					taskStatus.Result.Reason = err.Error()
					taskStatus.Result.ExitCode = -1
				}
			} else {
				fmt.Println("Task complete")
				taskStatus.Status = models.TaskStatusSuccess
				taskStatus.Result.ExitCode = 0
			}

			err = client.UpdateNodeTaskStatus(ctx, taskStatus)
			if err != nil {
				fmt.Printf("[Error] %s\n", err)
			}

		}
	}
}
