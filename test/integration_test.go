package test

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var dockerMongo = flag.Bool("docker-mongo", false, "start mongodb in a container")
var dockerCleanup = flag.Bool("docker-cleanup", true, "cleanup docker containers")
var pool *dockertest.Pool
var resources = []*dockertest.Resource{}

type MongoConnInfo struct {
	ConnectionString string
	DatabaseName     string
}

func TestMain(m *testing.M) {
	flag.Parse()

	if *dockerMongo {
		var err error
		pool, err = dockertest.NewPool("")
		if err != nil {
			Fatalf("Could not connect to docker: %s", err)
		}
	}

	connInfo := initializeMongo()

	clientOptions := options.Client().ApplyURI(connInfo.ConnectionString).SetDirect(true)
	c, err := mongo.NewClient(clientOptions)
	if err != nil {
		Fatalf("Failed to create mongo connection: %s", err)
	}

	if err := c.Connect(context.Background()); err != nil {
		Fatalf("failed to connect to mongo: %s", err)
	}

	defer c.Disconnect(context.Background()) // nolint: errcheck

	err = retry(60, time.Second, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
		defer cancel()
		if err := c.Ping(ctx, nil); err != nil {
			log.Printf("Failed to ping mongo: %s", err)
			return err
		}
		return nil
	})

	if err != nil {
		Fatalf("failed to connect to mongo: %s", err)
	}
}

func Fatalf(format string, v ...interface{}) {
	log.Printf(format, v...)
	cleanup()
	os.Exit(1)
}

func retry(tries int, backoff time.Duration, f func() error) error {
	for tries > 0 {
		err := f()
		if err == nil {
			return nil
		}
		tries--
		if tries > 0 {
			time.Sleep(backoff)
		} else {
			return err
		}
	}
	panic("unreachable")
}

func cleanup() {
	if pool != nil && *dockerCleanup {
		for _, resource := range resources {
			if err := pool.Purge(resource); err != nil {
				log.Printf("Failed to cleanup docker resource %s", resource.Container.ID)
			}
		}
	}
}

func initializeMongo() *MongoConnInfo {
	if pool != nil {
		resource, err := pool.Run("mongo", "3.6", nil)
		if err != nil {
			Fatalf("Could not start mongo: %s", err)
		}
		resources = append(resources, resource)
		return &MongoConnInfo{
			ConnectionString: fmt.Sprintf("mongodb://localhost:%s", resource.GetPort("27017/tcp")),
			DatabaseName:     "foodtruck-test",
		}
	} else {
		mongoConnString := os.Getenv("MONGODB_CONNECTION_STRING")
		mongoDBName := os.Getenv("MONGODB_DATABASE_NAME")

		if mongoConnString == "" || mongoDBName == "" {
			Fatalf("Running the tests requires mongodb. Either pass the -docker-mongo flag or set MONGODB_CONNECTION_STRING and MONGODB_DATABASE_NAME")
		}
		return &MongoConnInfo{
			ConnectionString: mongoConnString,
			DatabaseName:     mongoDBName,
		}
	}
}
