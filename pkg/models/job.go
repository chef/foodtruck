package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusExpired  TaskStatus = "expired"
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusComplete TaskStatus = "complete"
)

var ValidTaskStatuses = []string{
	string(TaskStatusPending),
	string(TaskStatusExpired),
	string(TaskStatusRunning),
	string(TaskStatusComplete),
}

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
	ID    string   `json:"id,omitempty" bson:"_id,omitempty"`
	Task  NodeTask `json:"task" bson:"task"`
	Nodes []Node   `json:"nodes" bson:"nodes,omitempty"`
}

type NodeTask struct {
	JobID       string          `json:"job_id,omitempty" bson:"job_id,omitempty"`
	WindowStart time.Time       `json:"window_start" bson:"window_start"`
	WindowEnd   time.Time       `json:"window_end" bson:"window_end"`
	Provider    string          `json:"provider" bson:"provider"`
	Spec        json.RawMessage `json:"spec" bson:"spec"`
}

type NodeTaskStatus struct {
	JobID      string     `json:"job_id" bson:"job_id"`
	Status     TaskStatus `json:"status,omitempty" bson:"status,omitempty"`
	LastUpdate time.Time  `json:"last_update,omitempty" bson:"last_update,omitempty"`
}
