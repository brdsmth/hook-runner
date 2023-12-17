package services

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

var DynamoClient *dynamodb.Client

// ConnectDynamoDB initializes and sets up a DynamoDB client.
func ConnectDynamoDB() {
	// Load the Shared AWS Configuration (~/.aws/config)
	/*
		When using the AWS SDK for Go, if you have set the environment variables as described above, the SDK will automatically use these credentials. You don't need to manually specify them in your code.
	*/
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithSharedConfigProfile("brdrsmth"),
		config.WithRegion("us-east-1"),
		// Uncomment and set the endpoint if you're using a local version of DynamoDB
		// config.WithEndpointResolver(aws.EndpointResolverFunc(
		//     func(service, region string) (aws.Endpoint, error) {
		//         if service == dynamodb.ServiceID {
		//             return aws.Endpoint{
		//                 URL:           "http://localhost:8000",
		//                 SigningRegion: "us-west-2",
		//             }, nil
		//         }
		//         // Fallback to default resolution
		//         return aws.Endpoint{}, &aws.EndpointNotFoundError{}
		//     })),
	)
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// fmt.Print(cfg)
	// Create DynamoDB client
	DynamoClient = dynamodb.NewFromConfig(cfg)

	// "Ping" DynamoDB to confirm connection
	tableName := "jobs" // Replace with your table name
	_, err = DynamoClient.DescribeTable(context.TODO(), &dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		log.Fatalf("Failed to connect to DynamoDB: %v", err)
	} else {
		log.Print("Connected to DynamodDB successfully")
	}
}
