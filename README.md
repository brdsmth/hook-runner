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

## Deployment

### Kubernetes

#### Local

Start `minikube`

```
minikube start
```

Direct `minikube` to use the `docker` env. Any `docker build ...` commands after this command is run will build inside the `minikube` registry and will not be visible in Docker Desktop. `minikube` uses its own docker daemon which is separate from the docker daemon on your host machine. Running `docker images` inside the `minikube` vm will show the images accessible to `minikube`

```
eval $(minikube docker-env)
```

```
docker build -t hook-runner-image:latest .
```

#### Environment Variables (if needed)

```
kubectl create secret generic awsconfig-secret --from-env-file=./.env
kubectl create secret generic rabbitmqurl-secret --from-env-file=./.env
kubectl create secret generic rabbitmqqueue-secret --from-env-file=./.env
kubectl create secret generic dynamodbqueuetable-secret --from-env-file=./.env
kubectl create secret generic dynamodbprocessedtable-secret --from-env-file=./.env
```

```
kubectl apply -f ./k8s/hook-runner.deployment.yaml
```

```
kubectl apply -f ./k8s/hook-runner.service.yaml
```

```
kubectl get deployments
```

```
kubectl get pods
```

```
minikube service hook-runner-service
```

After running the last comment the application will be able to be accessed in the browser at the specified port that `minikube` assigns.

#### Troubleshooting

```
minikube ssh 'docker images'
```

```
kubectl logs <pod-name>
```

```
kubectl logs -f <pod-name>
```
