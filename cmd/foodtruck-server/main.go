package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/chef/foodtruck/pkg/models"
	"github.com/chef/foodtruck/pkg/storage"
	"github.com/davecgh/go-spew/spew"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoDBConnectionStringEnvVarName = "MONGODB_CONNECTION_STRING"
	mongoDBDatabaseEnvVarName         = "MONGODB_DATABASE"
)

func main() {

	config := Config{
		Database:            "testdb1",
		JobsCollection:      "jobs",
		NodeTasksCollection: "node_tasks",
	}
	ctx := context.Background()
	c := connect()
	defer c.Disconnect(ctx)

	jobsCollection := c.Database(config.Database).Collection(config.JobsCollection)
	nodeTasksCollection := c.Database(config.Database).Collection(config.NodeTasksCollection)
	db := storage.CosmosDBImpl(jobsCollection, nodeTasksCollection)

	e := echo.New()
	e.Use(middleware.Logger())
	p := prometheus.NewPrometheus("foodtruck", nil)
	p.Use(e)
	initAdminRouter(e, db)
	initNodesRouter(e, db)

	e.Logger.Fatal(e.Start(":1323"))

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	err := db.AddJob(ctx, models.Job{
		Task: models.NodeTask{
			Type:        "infra",
			WindowStart: time.Now().AddDate(0, 1, 0),
			WindowEnd:   time.Now().AddDate(0, 1, 2),
		},
		Nodes: []models.Node{
			{
				Organization: "myorg",
				Name:         "testnode1",
			},
			{
				Organization: "myorg",
				Name:         "testnode2",
			},
			{
				Organization: "myorg",
				Name:         "testnode3",
			},
			{
				Organization: "myorg",
				Name:         "testnode4",
			},
		},
	})
	if err != nil {
		panic(err)
	}

	err = db.AddJob(ctx, models.Job{
		Task: models.NodeTask{
			Type:        "inspec",
			WindowStart: time.Now(),
			WindowEnd:   time.Now().AddDate(0, 0, 2),
		},
		Nodes: []models.Node{
			{
				Organization: "myorg",
				Name:         "testnode2",
			},
			{
				Organization: "myorg",
				Name:         "testnode3",
			},
		},
	})
	if err != nil {
		panic(err)
	}

	tasks, err := db.GetNodeTasks(ctx, models.Node{"myorg", "testnode2"})
	if err != nil {
		panic(err)
	}
	spew.Dump(tasks)

	task, err := db.NextNodeTask(ctx, models.Node{"myorg", "testnode2"})
	if err != nil {
		panic(err)
	}
	spew.Dump(task)

}

func connect() *mongo.Client {
	mongoDBConnectionString := os.Getenv(mongoDBConnectionStringEnvVarName)
	if mongoDBConnectionString == "" {
		log.Fatal("missing environment variable: ", mongoDBConnectionStringEnvVarName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	clientOptions := options.Client().ApplyURI(mongoDBConnectionString).SetDirect(true)
	c, err := mongo.NewClient(clientOptions)

	err = c.Connect(ctx)

	if err != nil {
		log.Fatalf("unable to initialize connection %v", err)
	}
	err = c.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("unable to connect %v", err)
	}
	return c
}
