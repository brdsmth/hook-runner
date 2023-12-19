package services

import (
	"context"
	localConfig "hook-runner/config"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var DynamoClient *dynamodb.Client

// ConnectDynamoDB initializes and sets up a DynamoDB client.
func ConnectDynamoDB() {
	/*
		When using the AWS SDK for Go, if you have set the environment variables in (~/.aws/config)
		the SDK will automatically use these credentials. You don't need to manually specify them in your code.
	*/
	// Load the Shared AWS Configuration (~/.aws/config)
	// The profile name will be stored in an environment variable
	awsConfigProfile := localConfig.ReadEnv("AWS_CONFIG_PROFILE")
	if awsConfigProfile == "" {
		log.Fatal("AWS_CONFIG_PROFILE environment variable not set")
	}
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile(awsConfigProfile),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// Create DynamoDB client
	DynamoClient = dynamodb.NewFromConfig(cfg)

	// Ping DynamoDB to confirm connection
	// The table name to store processed message will be stored in an environment variable
	dynamoDBProcessedTable := localConfig.ReadEnv("DYNAMODB_PROCESSED_TABLE")
	if dynamoDBProcessedTable == "" {
		log.Fatal("DYNAMODB_PROCESSED_TABLE environment variable not set")
	}
	tableName := dynamoDBProcessedTable
	_, err = DynamoClient.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		log.Fatalf("Failed to connect to DynamoDB: %v", err)
	} else {
		log.Print("Connected to DynamodDB successfully")
	}
}
