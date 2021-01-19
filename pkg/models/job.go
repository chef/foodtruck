package models

import (
	"encoding/json"
	"fmt"
	"time"
)

type TaskStatus string

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusComplete TaskStatus = "complete"
)

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
	Status      TaskStatus      `json:"status,omitempty" bson:"status,omitempty"`
	WindowStart time.Time       `json:"window_start" bson:"window_start"`
	WindowEnd   time.Time       `json:"window_end" bson:"window_end"`
	Type        string          `json:"type" bson:"type"`
	Spec        json.RawMessage `json:"spec" bson:"spec"`
}
