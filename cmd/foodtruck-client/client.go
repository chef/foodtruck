package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"syscall"
	"time"

	"github.com/chef/foodtruck/pkg/foodtruckhttp"
	"github.com/chef/foodtruck/pkg/models"
	"github.com/chef/foodtruck/pkg/provider"
)

type Duration time.Duration

func (d *Duration) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	td, err := time.ParseDuration(s)
	if err != nil {
		return err
	}

	*d = Duration(td)

	return nil
}

func (d Duration) MarshalJSON() (b []byte, err error) {
	return []byte(fmt.Sprintf(`"%s"`, time.Duration(d).String())), nil
}

type Config struct {
	Node          models.Node `json:"node"`
	AuthConfig    AuthConfig  `json:"auth"`
	BaseURL       string      `json:"base_url"`
	ProvidersPath string      `json:"providers_path"`
	Interval      Duration    `json:"interval"`
}

func (c Config) Validate() {
	fail := false
	if c.Node.Name == "" {
		fmt.Fprint(os.Stderr, "Node name must be provided\n")
		fail = true
	}

	if c.Node.Organization == "" {
		fmt.Fprint(os.Stderr, "Node org must be provided\n")
		fail = true
	}

	if c.BaseURL == "" {
		fmt.Fprint(os.Stderr, "Base URL must be provided\n")
		fail = true
	}

	if fail {
		os.Exit(1)
	}
}

func loadConfig(confPath string) Config {
	config := Config{
		BaseURL:  "http://localhost:1323",
		Interval: Duration(time.Second * 5),
	}

	if confPath != "" {
		d, err := ioutil.ReadFile(confPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read config: %v\n", err)
			os.Exit(1)
		}
		if err := json.Unmarshal(d, &config); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to parse config: %v\n", err)
			os.Exit(1)
		}
	}

	config.Validate()

	return config
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "[usage]: foodtruck-client conf.json\n")
		os.Exit(1)
	}

	config := loadConfig(os.Args[1])
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	fmt.Fprintf(os.Stderr, "Node %s checking into %s on interval %s\n", config.Node, config.BaseURL,
		time.Duration(config.Interval).String())

	authProvider, err := config.AuthConfig.AuthProvider.InitializeAuthProvider(config.Node.Name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize auth provider: %v\n", err)
		os.Exit(1)
	}

	client := foodtruckhttp.NewClient(config.BaseURL, config.Node, authProvider)
	runner := provider.NewExecRunner()
	for {
		select {
		case <-ctx.Done():
			break
		case <-time.After(time.Duration(config.Interval)):
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
