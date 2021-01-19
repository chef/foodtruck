package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/chef/foodtruck/pkg/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CosmosDB struct {
	jobsCollection           *mongo.Collection
	nodeTasksCollection      *mongo.Collection
	nodeTaskStatusCollection *mongo.Collection
}

type CosmosNodeTask struct {
	NodeName string            `bson:"node_name"`
	Tasks    []models.NodeTask `bson:"tasks"`
}

func CosmosDBImpl(jobsCollection *mongo.Collection, nodeTasksCollection *mongo.Collection, nodeTaskStatusCollection *mongo.Collection) Driver {
	return &CosmosDB{
		jobsCollection:           jobsCollection,
		nodeTasksCollection:      nodeTasksCollection,
		nodeTaskStatusCollection: nodeTaskStatusCollection,
	}
}

func (c *CosmosDB) AddJob(ctx context.Context, job models.Job) error {
	res, err := c.jobsCollection.InsertOne(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to insert job: %w", err)
	}

	job.Task.JobID = res.InsertedID.(primitive.ObjectID).Hex()

	updates := make([]mongo.WriteModel, len(job.Nodes))
	for i := range updates {
		nodeName := fmt.Sprintf("%s/%s", job.Nodes[i].Organization, job.Nodes[i].Name)
		updateModel := mongo.NewUpdateOneModel().SetFilter(
			bson.D{
				{"node_name", nodeName},
			},
		).SetUpdate(
			bson.D{
				{"$set", bson.D{{"node_name", nodeName}}},
				{"$push", bson.D{{"tasks", job.Task}}},
			},
		).SetUpsert(true)

		updates[i] = updateModel
	}

	opts := options.BulkWrite().SetOrdered(false)
	_, err = c.nodeTasksCollection.BulkWrite(ctx, updates, opts)

	if err != nil {
		return fmt.Errorf("failed to insert node_tasks: %w", err)
	}
	return nil
}

func (c *CosmosDB) ListJobs(ctx context.Context) error {
	return nil
}

func (c *CosmosDB) GetNodeTasks(ctx context.Context, node models.Node) ([]models.NodeTask, error) {
	cursor := c.nodeTasksCollection.FindOne(ctx, bson.D{{"node_name", node.String()}})
	if err := cursor.Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, models.ErrNoTasks
		}
		return nil, fmt.Errorf("failed to query for node tasks: %w", err)
	}
	var result CosmosNodeTask
	if err := cursor.Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode node tasks result: %w", err)
	}
	return result.Tasks, nil
}

func (c *CosmosDB) NextNodeTask(ctx context.Context, node models.Node) (models.NodeTask, error) {
	cursor := c.nodeTasksCollection.FindOne(ctx, bson.D{{"node_name", node.String()}})
	if err := cursor.Err(); err != nil {
		return models.NodeTask{}, fmt.Errorf("failed to query for node tasks: %w", err)
	}
	var result CosmosNodeTask
	if err := cursor.Decode(&result); err != nil {
		return models.NodeTask{}, fmt.Errorf("failed to decode node tasks result: %w", err)
	}

	tasks := result.Tasks

	for {
		if len(tasks) == 0 {
			return models.NodeTask{}, models.ErrNoTasks
		}

		nextTask := tasks[0]
		for i := 1; i < len(tasks); i++ {
			if tasks[i].WindowStart.Before(nextTask.WindowStart) {
				nextTask = tasks[i]
			}
		}

		if time.Now().After(nextTask.WindowStart) && time.Now().Before(nextTask.WindowEnd) {
			if err := c.dequeueTask(ctx, node, nextTask.JobID, models.TaskStatusPending); err != nil {
				return models.NodeTask{}, fmt.Errorf("failed to remove task: %w", err)
			}
			return nextTask, nil
		} else if time.Now().After(nextTask.WindowEnd) {
			if err := c.dequeueTask(ctx, node, nextTask.JobID, "expired"); err != nil {
				return models.NodeTask{}, fmt.Errorf("failed to remove task: %w", err)
			}
		}
		tasks = tasks[1:]
	}
}

func (c *CosmosDB) dequeueTask(ctx context.Context, node models.Node, jobID string, status models.TaskStatus) error {
	updates := make([]mongo.WriteModel, 1)
	nodeName := fmt.Sprintf("%s/%s", node.Organization, node.Name)
	updateNodeTasksModel := mongo.NewUpdateOneModel().SetFilter(
		bson.D{
			{"node_name", nodeName},
		},
	).SetUpdate(
		bson.D{
			{"$set", bson.D{{"node_name", nodeName}}},
			{"$pull", bson.D{{"tasks", bson.D{{"job_id", jobID}}}}},
		},
	).SetUpsert(true)

	updates[0] = updateNodeTasksModel

	opts := options.BulkWrite().SetOrdered(false)
	_, err := c.nodeTasksCollection.BulkWrite(ctx, updates, opts)

	if err != nil {
		return err
	}

	err = c.UpdateNodeTaskStatus(ctx, node, models.NodeTaskStatus{
		JobID:  jobID,
		Status: models.TaskStatusPending,
	})
	if err != nil {
		// TODO: logging
		fmt.Printf("failed to create task status: %s\n", err)
	}

	return nil
}

func (c *CosmosDB) UpdateNodeTaskStatus(ctx context.Context, node models.Node, nodeTaskStatus models.NodeTaskStatus) error {
	nodeName := node.String()

	opts := options.Update().SetUpsert(true)
	_, err := c.nodeTaskStatusCollection.UpdateOne(
		ctx,
		bson.D{
			{"node_name", nodeName},
			{"job_id", nodeTaskStatus.JobID},
		},
		bson.D{
			{"$set", bson.D{
				{"status", nodeTaskStatus.Status},
				{"last_updated", time.Now()},
				{"node_name", nodeName},
				{"job_id", nodeTaskStatus.JobID},
				{"result", nodeTaskStatus.Result},
			}},
		},
		opts,
	)
	if err != nil {
		return fmt.Errorf("failed to update node task status: %w", err)
	}
	return nil
}
