package main

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CreateCollectionCommand struct {
	CustomAction string `bson:"customAction"`
	Collection   string `bson:"collection"`
	ShardKey     string `bson:"shardKey"`
}

const (
	namespaceExistsErrCode int32 = 48
)

func initialzeCollections(ctx context.Context, db *mongo.Database) {
	err := createCollection(ctx, db, "jobs", "_id")
	if err != nil {
		fmt.Printf("Error creating collection(jobs): %v\n", err)
	}

	err = createCollection(ctx, db, "node_tasks", "node_name")
	if err != nil {
		fmt.Printf("Error creating collection(node_tasks): %v\n", err)
	}

	err = createCollection(ctx, db, "node_task_status", "node_name", "job_id")
	if err != nil {
		fmt.Printf("Error creating collection(node_name): %v\n", err)
	}
}

func createCollection(ctx context.Context, db *mongo.Database, collectionName string, shardKey string, indexes ...string) error {
	res := db.RunCommand(ctx,
		CreateCollectionCommand{
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
	indexOpts.SetUnique(true)
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
