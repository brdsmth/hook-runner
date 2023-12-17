package services

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var MongoClient *mongo.Client

// ConnectDB initializes and returns a new DB object with a MongoDB client.
func ConnectMongoDB(mongoDBURL string) {
	var err error

	serverAPI := options.ServerAPI(options.ServerAPIVersion1)
	opts := options.Client().ApplyURI(mongoDBURL).SetServerAPIOptions(serverAPI)

	MongoClient, err = mongo.Connect(context.Background(), opts)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Ping MongoDB to confirm connection
	if err := MongoClient.Database("admin").RunCommand(context.Background(), bson.D{{Key: "ping", Value: 1}}).Err(); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}
	log.Print("Connected to MongoDB successfully.")
}

func ColHelper(collectionName string) *mongo.Collection {
	if MongoClient == nil {
		log.Fatalf("MongoDB client is not initialized")
	}
	return MongoClient.Database("hooksms").Collection(collectionName)
}

// Get jobs due from the database
func GetDueJobs() ([]*Job, error) {
	collection := ColHelper("queue")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	var jobs []*Job
	defer cancel()

	res, err := collection.Find(ctx, bson.M{})

	if err != nil {
		return nil, err
	}

	defer res.Close(ctx)
	for res.Next(ctx) {
		var job *Job
		if err = res.Decode(&job); err != nil {
			log.Fatal(err)
		}
		jobs = append(jobs, job)
	}

	return jobs, err
}
