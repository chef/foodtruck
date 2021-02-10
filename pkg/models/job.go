package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending TaskStatus = "pending"
	TaskStatusRunning TaskStatus = "running"
	TaskStatusFailed  TaskStatus = "failed"
	TaskStatusSuccess TaskStatus = "success"
)

var ValidTaskStatuses = []string{
	string(TaskStatusPending),
	string(TaskStatusRunning),
	string(TaskStatusFailed),
	string(TaskStatusSuccess),
}

type JobID = string

func IsValidTaskStatus(s string) bool {
	for i := range ValidTaskStatuses {
		if s == ValidTaskStatuses[i] {
			return true
		}
	}
	return false
}

type Node struct {
	Organization string `json:"org" bson:"org"`
	Name         string `json:"name" bson:"name"`
}

func (n Node) String() string {
	return fmt.Sprintf("%s/%s", n.Organization, n.Name)
}

type Job struct {
	ID    JobID    `json:"id,omitempty" bson:"_id,omitempty"`
	Task  NodeTask `json:"task" bson:"task"`
	Nodes []Node   `json:"nodes" bson:"nodes,omitempty"`
}

type NodeTask struct {
	JobID       JobID           `json:"job_id,omitempty" bson:"job_id,omitempty"`
	WindowStart time.Time       `json:"window_start" bson:"window_start"`
	WindowEnd   time.Time       `json:"window_end" bson:"window_end"`
	Provider    string          `json:"provider" bson:"provider"`
	Spec        json.RawMessage `json:"spec" bson:"spec"`
}

type NodeTaskStatusResult struct {
	ExitCode int    `json:"exit_code" bson:"exit_code"`
	Reason   string `json:"reason,omitempty" bson:"reason,omitempty"`
}

type NodeTaskStatus struct {
	JobID       JobID                 `json:"job_id" bson:"job_id"`
	NodeName    string                `json:"node_name" bson:"node_name"`
	Status      TaskStatus            `json:"status,omitempty" bson:"status,omitempty"`
	LastUpdated time.Time             `json:"last_updated,omitempty" bson:"last_updated,omitempty"`
	Result      *NodeTaskStatusResult `json:"result,omitempty" bson:"result,omitempty"`
}
