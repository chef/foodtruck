package storage

import (
	"context"

	"github.com/chef/foodtruck/pkg/models"
)

type JobWithStatus struct {
	Job      models.Job              `json:"job"`
	Statuses []models.NodeTaskStatus `json:"statuses,omitempty"`
}

type GetJobOpts struct {
	FetchStatuses bool
}

type GetJobOpt func(*GetJobOpts)

func WithJobStatuses(fetchStatus bool) GetJobOpt {
	return func(opts *GetJobOpts) {
		opts.FetchStatuses = fetchStatus
	}
}

type Driver interface {
	AddJob(ctx context.Context, job models.Job) (models.JobID, error)
	ListJobs(ctx context.Context) error
	GetJob(ctx context.Context, jobID models.JobID, opts ...GetJobOpt) (JobWithStatus, error)
	GetNodeTasks(ctx context.Context, node models.Node) ([]models.NodeTask, error)
	NextNodeTask(ctx context.Context, node models.Node) (models.NodeTask, error)
	UpdateNodeTaskStatus(ctx context.Context, node models.Node, nodeTaskStatus models.NodeTaskStatus) error
}
