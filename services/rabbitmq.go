package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	amqp "github.com/rabbitmq/amqp091-go"
)

var RabbitMQConn *amqp.Connection
var RabbitMQConnMutex sync.Mutex

type Job struct {
	ID        string      `json:"id"`
	Payload   interface{} `json:"payload"`
	URL       string      `json:"url"`
	Timestamp string      `json:"timestamp"`
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
	q, err := ch.QueueDeclare("QUEUE", false, false, false, false, nil)
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
		for d := range msgs {
			processMessage(d) // Or, for concurrent processing of each message: go processMessage(d)
		}
	}()

	// go func() {
	// 	for d := range msgs {
	// 		url := string(d.Body)

	// 		// Make HTTP request
	// 		resp, err := http.Get(url)
	// 		if err != nil {
	// 			log.Printf("Error making HTTP request: %v", err)
	// 			continue
	// 		}
	// 		resp.Body.Close() // Close the response body

	// 		// Update MongoDB
	// 		filter := bson.M{"url": url}
	// 		update := bson.M{"$set": bson.M{"status": "processed"}}
	// 		_, err = MongoClient.Database("your_db").Collection("your_collection").UpdateOne(context.Background(), filter, update)
	// 		if err != nil {
	// 			log.Printf("Failed to update MongoDB: %v", err)
	// 		}

	// 		// Check if job is recurrent and requeue if necessary
	// 		if isRecurrentJob(url) {
	// 			err = ch.PublishWithContext(context.Background(), "", "QUEUE", false, false, amqp.Publishing{
	// 				ContentType: "text/plain",
	// 				Body:        []byte(url),
	// 			})
	// 			if err != nil {
	// 				log.Printf("Failed to requeue job: %v", err)
	// 			}
	// 		}
	// 	}
	// }()

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
	payload := string(msg.Body)
	jobURL := job.URL
	jobURLString := string(job.URL)

	log.Print(payload)
	log.Print(jobURL)
	log.Print(jobURLString)

	// Make HTTP request
	resp, err := http.Get(job.URL)
	if err != nil {
		log.Printf("Error making HTTP request: %v", err)
		return
	}
	resp.Body.Close() // Close the response body

	// Update DynamoDB
	partitionKey := fmt.Sprintf("job:%s:%s", job.ID, job.Timestamp)

	// Marshal job.Payload to a JSON string
	jsonPayload, err := json.Marshal(job.Payload)
	if err != nil {
		log.Printf("Error marshalling payload to JSON: %v", err)
		return
	}

	// Prepare the item to write to DynamoDB
	item := map[string]types.AttributeValue{
		"RowKey":  &types.AttributeValueMemberS{Value: partitionKey},
		"Payload": &types.AttributeValueMemberS{Value: string(jsonPayload)},
		// Include other fields as necessary
	}

	// Write to DynamoDB
	_, err = DynamoClient.PutItem(context.Background(), &dynamodb.PutItemInput{
		TableName: aws.String("jobs"),
		Item:      item,
	})
	if err != nil {
		log.Printf("Failed to write to DynamoDB: %v", err)
		return
	}

	log.Printf("Successfully wrote job with URL %s to DynamoDB", job.URL)
}

func isRecurrentJob(url string) bool {
	// Implement your logic to check if the job is recurrent
	return false
}
