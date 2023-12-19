package services

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	localConfig "hook-runner/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

var RabbitMQConn *amqp.Connection
var RabbitMQConnMutex sync.Mutex

type Job struct {
	RowKey    string      `json:"RowKey"`
	JobID     string      `json:"JobID"`
	Payload   interface{} `json:"Payload"`
	URL       string      `json:"URL"`
	ExecuteAt string      `json:"ExecuteAt"`
	Status    string      `json:"Status"`
}

func ConnectToRabbitMQ(rabbitMQURL string) {
	var err error
	RabbitMQConn, err = amqp.Dial(rabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	log.Println("Connected to RabbitMQ successfully")
}

func Fetch() {
	// Load RabbitMQ
	RabbitMQConnMutex.Lock()
	defer RabbitMQConnMutex.Unlock()

	if RabbitMQConn == nil {
		log.Println("RabbitMQ connection is not available")
		return
	}

	// Create a channel
	ch, err := RabbitMQConn.Channel()
	if err != nil {
		log.Printf("Failed to open a channel: %v", err)
		return
	}
	defer ch.Close()

	// Declare the queue
	// Pull the name of the RabbitMQ queue from env
	rabbitMQQueue := localConfig.ReadEnv("RABBITMQ_QUEUE")
	if rabbitMQQueue == "" {
		log.Fatal("RABBITMQ_QUEUE environment variable not set")
	}
	q, err := ch.QueueDeclare(rabbitMQQueue, false, false, false, false, nil)
	if err != nil {
		log.Printf("Failed to declare a queue: %v", err)
		return
	}

	// Consume messages
	msgs, err := ch.Consume(q.Name, "", true, false, false, false, nil)
	if err != nil {
		log.Printf("Failed to register a consumer: %v", err)
		return
	}

	forever := make(chan bool)

	/*
		Asynchronous Message Processing:

		RabbitMQ can send messages to your consumer (the Go program) at a high rate. Handling each message synchronously (one after the other) could potentially lead to delays, especially if processing each message (like making an HTTP request) is time-consuming.
		By using a goroutine for message processing, you allow your program to start processing a new message as soon as it's received, without waiting for the previous message's processing to complete. This is particularly beneficial for I/O-bound tasks like HTTP requests.
		Why Declared Before the For Loop:

		The goroutine is declared before the for loop to ensure that the message consumption starts concurrently with the main program. As soon as the program sets up the RabbitMQ consumer and starts receiving messages, the goroutine begins processing them.
		If you were to start the goroutine inside the for loop, it would create a new goroutine for each message, which might not be necessary and could lead to a large number of goroutines, potentially overwhelming the system.
	*/
	go func() {
		for m := range msgs {
			/*
				For concurrent processing of each message: go processMessage(d)
			*/
			processMessage(m)
		}
	}()

	log.Printf(" [*] Waiting for messages. To exit press CTRL+C")
	/*
		The forever channel and the <-forever line are a simple way to keep your main function active while other goroutines are doing their work. This pattern is common in Go programs that start background tasks (like processing messages from a queue) and need to keep running indefinitely to allow those tasks to continue.
	*/
	<-forever
}

func processMessage(msg amqp.Delivery) {
	// Parse the JSON message
	var job Job
	err := json.Unmarshal(msg.Body, &job)
	if err != nil {
		log.Printf("Error parsing JSON message: %v", err)
		return
	}

	// DEBUG	-> print out the job grabbed from the RabbitMQ queue
	// log.Print(job)
	log.Printf("Processing:\t\t%s", job.RowKey)

	var payloadReader io.Reader

	switch payload := job.Payload.(type) {
	case string:
		// If payload is already a JSON string, use it directly
		log.Println("payload already a json string")
		payloadReader = strings.NewReader(payload)
	default:
		log.Println("payload not a json string")
		// Else, marshal the payload to JSON
		payloadBytes, err := json.Marshal(job.Payload)
		if err != nil {
			log.Printf("Error marshaling payload for job %s: %v", job.JobID, err)
			return
		}
		payloadReader = bytes.NewReader(payloadBytes)
	}

	// Make an HTTP POST request with the JSON payload
	resp, err := http.Post(job.URL, "application/json", payloadReader)
	if err != nil {
		log.Printf("Error making HTTP POST request to %s: %v", job.URL, err)
	}
	log.Printf("Sent: %s to URL: %s with Status: %s", job.JobID, job.URL, resp.Status)

	// Update DynamoDB
	// Set the status of the job after the POST request is made
	job.Status = "PROCESSED"

	// Marshal the Payload to a JSON string
	postProcessPayloadBytes, err := json.Marshal(job.Payload)
	if err != nil {
		log.Printf("Error procesing job:\t\t%s", job.RowKey)
		return
	}
	postProcessPayloadString := string(postProcessPayloadBytes)
	currentTime := time.Now().Format(time.RFC3339)
	item := map[string]types.AttributeValue{
		"RowKey":     &types.AttributeValueMemberS{Value: job.RowKey},
		"JobID":      &types.AttributeValueMemberS{Value: job.JobID},
		"Payload":    &types.AttributeValueMemberS{Value: postProcessPayloadString},
		"URL":        &types.AttributeValueMemberS{Value: job.URL},
		"Status":     &types.AttributeValueMemberS{Value: job.Status},
		"ExecutedAt": &types.AttributeValueMemberS{Value: currentTime},
	}

	// Write the update job to the `processed` Table
	// The table name to store processed message will be stored in an environment variable
	dynamoDBProcessedTable := localConfig.ReadEnv("DYNAMODB_PROCESSED_TABLE")
	if dynamoDBProcessedTable == "" {
		log.Fatal("DYNAMODB_PROCESSED_TABLE environment variable not set")
	}
	processedTableName := dynamoDBProcessedTable
	_, err = DynamoClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String(processedTableName),
		Item:      item,
	})
	if err != nil {
		log.Printf("Failed to write to DynamoDB: %v", err)
		return
	}

	// Delete the job from the queue table
	// The table name to store processed message will be stored in an environment variable
	dynamoDBQueueTable := localConfig.ReadEnv("DYNAMODB_QUEUE_TABLE")
	if dynamoDBQueueTable == "" {
		log.Fatal("DYNAMODB_QUEUE_TABLE environment variable not set")
	}
	queueTableName := dynamoDBQueueTable
	_, err = DynamoClient.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
		TableName: aws.String(queueTableName),
		Key: map[string]types.AttributeValue{
			"RowKey":    &types.AttributeValueMemberS{Value: job.RowKey},
			"ExecuteAt": &types.AttributeValueMemberS{Value: job.ExecuteAt},
		},
	})
	if err != nil {
		log.Printf("Failed to delete from DynamoDB: %v, %s", err, job.RowKey)
		return
	}

	log.Printf("Processed job: %s with execute time: %s at %s", job.JobID, job.ExecuteAt, currentTime)
}
