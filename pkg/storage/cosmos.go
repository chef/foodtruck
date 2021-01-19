package storage

import (
	"context"
	"fmt"

	"github.com/chef/foodtruck/pkg/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CosmosDB struct {
	jobsCollection      *mongo.Collection
	nodeTasksCollection *mongo.Collection
}

type CosmosNodeTask struct {
	NodeName string            `bson:"node"`
	Tasks    []models.NodeTask `bson:"tasks"`
}

func CosmosDBImpl(jobsCollection *mongo.Collection, nodeTasksCollection *mongo.Collection) Driver {
	return &CosmosDB{
		jobsCollection:      jobsCollection,
		nodeTasksCollection: nodeTasksCollection,
	}
}

func (c *CosmosDB) AddJob(ctx context.Context, job models.Job) error {
	res, err := c.jobsCollection.InsertOne(ctx, job)
	if err != nil {
		return fmt.Errorf("failed to insert job: %w", err)
	}

	job.Task.JobID = res.InsertedID.(primitive.ObjectID).Hex()
	job.Task.Status = models.TaskStatusPending

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
	_, err = c.nodeTasksCollection.BulkWrite(context.TODO(), updates, opts)

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
		return nil, fmt.Errorf("failed to query for node tasks: %w", err)
	}
	var result CosmosNodeTask
	if err := cursor.Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode node tasks result: %w", err)
	}
	return result.Tasks, nil
}
