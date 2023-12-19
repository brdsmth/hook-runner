## Introduction

The `hook-runner` service exists to fetch messages from the **RabbitMQ** distributed message queue and **POST** these messages to a separate server specified by each job in the queue. The option to include a different URL for each job makes this distributed job queue system robust. Each message roughly looks like the below:

```javascript
    {
        "JobID": "O91ied",
        "URL": "http://localhost:8080/test 2023-12-16T22:36:10Z",
        "ExecuteAt": "2023-12-16T22:36:10Z"
        "Payload": {
            "key1": "example",
            "key2": "data",
            "key3": "passed",
            "key4": "to",
            "key5": "url"
        },
    }
```

Every time the `hook-runner` grabs a message from the queue it will make a **POST** request to the specified URL and include the payload as the body of the **POST** request. The `hook-runner` will then move the job from the **queue** table of **DynamoDB** to the **processed** table and update the status of the job accordingly.

### Usage

The `hook-runner` service is meant to be run in conjunction with the `hook-api` and the `hook-scheduler` services.

To run the `hook-runner` service the following `.env` variables need to be set

```
AWS_CONFIG_PROFILE=
RABBITMQ_URL=
RABBITMQ_QUEUE=
DYNAMODB_QUEUE_TABLE=
DYNAMODB_PROCESSED_TABLE=
```

Once these are added the server can be started by

```
go run main.go
```

Once the `hook-runner` service has started it will start listening on the `RABBITMQ_URL` for messages added to the `RABBITMQ_QUEUE`.
