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

type createCollectionCommand struct {
	CustomAction string `bson:"customAction"`
	Collection   string `bson:"collection"`
	ShardKey     string `bson:"shardKey"`
}

const (
	namespaceExistsErrCode int32 = 48
)

func initialzeCollections(ctx context.Context, db *mongo.Database) error {
	err := createCollection(ctx, db, "jobs", "_id", true)
	if err != nil {
		return fmt.Errorf("failed creating collection(jobs): %w", err)
	}

	err = createCollection(ctx, db, "node_tasks", "node_name", true)
	if err != nil {
		return fmt.Errorf("failed creating collection(node_tasks): %w", err)
	}

	err = createCollection(ctx, db, "node_task_status", "node_name", false, "job_id")
	if err != nil {
		return fmt.Errorf("failed creating collection(node_name): %w", err)
	}
	return nil
}

func createCollection(ctx context.Context, db *mongo.Database, collectionName string, shardKey string, shardKeyUnique bool,
	indexes ...string) error {
	res := db.RunCommand(ctx,
		createCollectionCommand{
			CustomAction: "CreateCollection",
			Collection:   collectionName,
			ShardKey:     shardKey,
		})
	if err := res.Err(); err != nil {
		cmdErr, ok := err.(mongo.CommandError)
		if !ok || cmdErr.Code != namespaceExistsErrCode {
			return err
		}
	}

	indexView := db.Collection(collectionName).Indexes()

	indexOpts := &options.IndexOptions{}
	if shardKeyUnique {
		indexOpts.SetUnique(true)
	}

	_, err := indexView.CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{shardKey, 1}},
		Options: indexOpts,
	})

	if err != nil {
		cmdErr, ok := err.(mongo.CommandError)
		if !ok || cmdErr.Code != namespaceExistsErrCode {
			return err
		}
	}

	for i := range indexes {
		_, err := indexView.CreateOne(ctx, mongo.IndexModel{
			Keys: bson.D{{indexes[i], 1}},
		})

		if err != nil {
			cmdErr, ok := err.(mongo.CommandError)
			if !ok || cmdErr.Code != namespaceExistsErrCode {
				return err
			}
		}
	}

	return nil
}

func InitCosmosDB(ctx context.Context, c *mongo.Client, databaseName string) (*CosmosDB, error) {

	db := c.Database(databaseName)

	if err := initialzeCollections(ctx, c.Database(databaseName)); err != nil {
		return nil, err
	}

	jobsCollection := db.Collection("jobs")
	nodeTasksCollection := db.Collection("node_tasks")
	nodeTaskStatusCollection := db.Collection("node_task_status")

	return CosmosDBImpl(jobsCollection, nodeTasksCollection, nodeTaskStatusCollection), nil
}

func InitMongoDB(ctx context.Context, c *mongo.Client, databaseName string) (*CosmosDB, error) {
	db := c.Database(databaseName)

	jobsCollection := db.Collection("jobs")
	nodeTasksCollection := db.Collection("node_tasks")
	nodeTaskStatusCollection := db.Collection("node_task_status")

	return CosmosDBImpl(jobsCollection, nodeTasksCollection, nodeTaskStatusCollection), nil
}

func CosmosDBImpl(jobsCollection *mongo.Collection, nodeTasksCollection *mongo.Collection, nodeTaskStatusCollection *mongo.Collection) *CosmosDB {
	return &CosmosDB{
		jobsCollection:           jobsCollection,
		nodeTasksCollection:      nodeTasksCollection,
		nodeTaskStatusCollection: nodeTaskStatusCollection,
	}
}

func (c *CosmosDB) AddJob(ctx context.Context, job models.Job) (models.JobID, error) {
	res, err := c.jobsCollection.InsertOne(ctx, job)
	if err != nil {
		return "", fmt.Errorf("failed to insert job: %w", err)
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
		return "", fmt.Errorf("failed to insert node_tasks: %w", err)
	}
	return job.Task.JobID, nil
}

func (c *CosmosDB) ListJobs(ctx context.Context) error {
	return nil
}

func (c *CosmosDB) GetJob(ctx context.Context, jobID models.JobID, opts ...GetJobOpt) (JobWithStatus, error) {
	gopts := GetJobOpts{}
	for _, o := range opts {
		o(&gopts)
	}

	objID, err := primitive.ObjectIDFromHex(jobID)
	if err != nil {
		return JobWithStatus{}, models.ErrNotFound
	}

	cursor := c.jobsCollection.FindOne(ctx, bson.D{{"_id", objID}})
	if err := cursor.Err(); err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return JobWithStatus{}, models.ErrNotFound
		}
		return JobWithStatus{}, fmt.Errorf("failed to query for jobs: %w", err)
	}

	job := models.Job{}
	if err := cursor.Decode(&job); err != nil {
		return JobWithStatus{}, err
	}

	var nodeStatuses []models.NodeTaskStatus
	if gopts.FetchStatuses {
		cursor, err := c.nodeTaskStatusCollection.Find(ctx, bson.D{{"job_id", jobID}})
		if err != nil {
			if errors.Is(err, mongo.ErrNoDocuments) {
				return JobWithStatus{}, models.ErrNotFound
			}
			return JobWithStatus{}, fmt.Errorf("failed to query for jobs: %w", err)
		}
		if err := cursor.All(ctx, &nodeStatuses); err != nil {
			return JobWithStatus{}, err
		}
	}

	return JobWithStatus{
		Job:      job,
		Statuses: nodeStatuses,
	}, nil
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
		if errors.Is(err, mongo.ErrNoDocuments) {
			return models.NodeTask{}, models.ErrNoTasks
		}
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
