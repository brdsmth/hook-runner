package main

import (
	"hook-runner/config"
	"hook-runner/services"
	"log"
	"net/http"
)

func main() {
	var err error

	// Connect to MongoDB
	mongoDBURL := config.ReadEnv("MONGODB_URI")
	if mongoDBURL == "" {
		log.Fatal("MONGODB_URI environment variable not set")
	}
	services.ConnectMongoDB(mongoDBURL)

	// Connect to RabbitMQ
	// rabbitMQURL := "amqp://guest:guest@localhost:5672/" // local rabbitmq url
	rabbitMQURL := config.ReadEnv("RABBITMQ_URL")
	if rabbitMQURL == "" {
		log.Fatal("RABBITMQ_URL environment variable not set")
	}

	services.ConnectToRabbitMQ(rabbitMQURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer services.RabbitMQConn.Close()

	// Initiate cron
	// services.RunCron()

	// Fetch messages
	services.Fetch()

	// Define routes and handlers for sending SMS messages
	// http.HandleFunc("/queue-sms", handlers.QueueSMSHandler)
	// http.HandleFunc("/schedule-sms", handlers.ScheduleSMSHandler)

	// Start the HTTP server for the publisher microservice
	log.Println("Server listening on 8080")
	http.ListenAndServe(":8080", nil)
}
