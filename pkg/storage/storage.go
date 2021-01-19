package storage

import (
	"context"

	"github.com/chef/foodtruck/pkg/models"
)

type Driver interface {
	AddJob(ctx context.Context, job models.Job) error
	ListJobs(ctx context.Context) error
	GetNodeTasks(ctx context.Context, node models.Node) ([]models.NodeTask, error)
	NextNodeTask(ctx context.Context, node models.Node) (models.NodeTask, error)
	UpdateNodeTaskStatus(ctx context.Context, node models.Node, nodeTaskStatus models.NodeTaskStatus) error
}
