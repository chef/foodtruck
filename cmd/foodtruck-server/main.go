package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/chef/foodtruck/pkg/server"
	"github.com/chef/foodtruck/pkg/storage"
	"github.com/labstack/echo-contrib/prometheus"
	"github.com/labstack/echo/v4/middleware"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoDBConnectionStringEnvVarName = "MONGODB_CONNECTION_STRING"
	mongoDBDatabaseNameEnvVarName     = "MONGODB_DATABASE_NAME"
	nodesAPIKeyEnvVarName             = "NODES_API_KEY"
	adminAPIKeyEnvVarName             = "ADMIN_API_KEY"
	foodtruckPortEnvVarName           = "FOODTRUCK_LISTEN_ADDR"
)

type Config struct {
	ListenAddr         string
	DatabaseConnection string
	Database           string
	Auth               struct {
		// Auth for the nodes endpoints
		Nodes struct {
			ApiKey string
		}

		// Auth for the admin endpoints
		Admin struct {
			ApiKey string
		}
	}
}

func loadConfig() Config {
	c := Config{}
	{
		v, ok := os.LookupEnv(foodtruckPortEnvVarName)
		if !ok {
			v = ":1323"
		}
		c.ListenAddr = v
	}

	{
		v, ok := os.LookupEnv(mongoDBConnectionStringEnvVarName)
		if !ok {
			fmt.Fprintf(os.Stderr, "You must provide %s in the environment\n", mongoDBConnectionStringEnvVarName)
			os.Exit(1)
		}
		c.DatabaseConnection = v
	}

	{
		v, ok := os.LookupEnv(mongoDBDatabaseNameEnvVarName)
		if !ok {
			fmt.Fprintf(os.Stderr, "You must provide %s in the environment\n", mongoDBDatabaseNameEnvVarName)
			os.Exit(1)
		}
		c.Database = v
	}

	{
		v, ok := os.LookupEnv(nodesAPIKeyEnvVarName)
		if !ok {
			fmt.Fprintf(os.Stderr, "You must provide %s in the environment\n", nodesAPIKeyEnvVarName)
			os.Exit(1)
		}
		c.Auth.Nodes.ApiKey = v
	}

	{
		v, ok := os.LookupEnv(adminAPIKeyEnvVarName)
		if !ok {
			fmt.Fprintf(os.Stderr, "You must provide %s in the environment\n", adminAPIKeyEnvVarName)
			os.Exit(1)
		}
		c.Auth.Admin.ApiKey = v
	}
	return c
}

func main() {
	config := loadConfig()

	ctx := context.Background()
	c := connect()
	defer c.Disconnect(ctx)

	db, err := storage.InitCosmosDB(ctx, c, config.Database)
	if err != nil {
		log.Fatalf("failed to initialize cosmos backend: %s", err)
	}

	e := server.Setup(db, config.Auth.Admin.ApiKey, config.Auth.Nodes.ApiKey)
	e.Use(middleware.Logger())
	p := prometheus.NewPrometheus("foodtruck", nil)
	p.Use(e)

	e.Logger.Fatal(e.Start(config.ListenAddr))
}

func connect() *mongo.Client {
	mongoDBConnectionString := os.Getenv(mongoDBConnectionStringEnvVarName)
	if mongoDBConnectionString == "" {
		log.Fatal("missing environment variable: ", mongoDBConnectionStringEnvVarName)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	clientOptions := options.Client().ApplyURI(mongoDBConnectionString).SetDirect(true)
	c, err := mongo.NewClient(clientOptions)

	err = c.Connect(ctx)

	if err != nil {
		log.Fatalf("unable to initialize connection %v", err)
	}
	err = c.Ping(ctx, nil)
	if err != nil {
		log.Fatalf("unable to connect to mongodb %v", err)
	}
	return c
}
