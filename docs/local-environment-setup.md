# Stathera: Local Environment Setup Guide

This comprehensive guide will walk you through setting up, running, and testing the Stathera financial platform in a local environment. Stathera is a Go-based system that provides transaction processing, order book management, and supply management capabilities.

## Table of Contents

1. [Setting Up Your Development Environment](#1-setting-up-your-development-environment)
2. [Setting Up Infrastructure Dependencies](#2-setting-up-infrastructure-dependencies)
3. [Configuring the Application](#3-configuring-the-application)
4. [Running the Application](#4-running-the-application)
5. [Testing the Application](#5-testing-the-application)
6. [Implementing New Features](#6-implementing-new-features)
7. [Troubleshooting](#7-troubleshooting)
8. [Monitoring](#8-monitoring)

## 1. Setting Up Your Development Environment

### 1.1 Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.24 or later**
  ```bash
  # Check Go version
  go version
  
  # If not installed or outdated, download from https://golang.org/dl/
  ```

- **Git**
  ```bash
  # Check Git version
  git version
  
  # If not installed, follow instructions at https://git-scm.com/downloads
  ```

- **Docker and Docker Compose** (for running Redis and Kafka)
  ```bash
  # Check Docker version
  docker --version
  docker-compose --version
  
  # If not installed, follow instructions at https://docs.docker.com/get-docker/
  ```

### 1.2 Clone the Repository

```bash
# Clone the repository
git clone https://github.com/cmatc13/stathera.git
cd stathera
```

### 1.3 Install Dependencies

```bash
# Install Go dependencies
go mod download

# Alternatively, use the provided script
./scripts/download-deps.sh
```

## 2. Setting Up Infrastructure Dependencies

Stathera requires Redis and Kafka to run. The easiest way to set these up is using Docker Compose.

### 2.1 Start Redis and Kafka

```bash
# Start Redis and Kafka using Docker Compose
docker-compose up -d
```

This command starts:
- Redis on port 6379
- Zookeeper on port 2181
- Kafka on port 9092

### 2.2 Verify Services are Running

```bash
# Check if containers are running
docker-compose ps

# Check Redis connection
#redis-cli ping

e.g. 

# Check Kafka topics (optional, requires Kafka tools)
kafka-topics.sh --bootstrap-server localhost:9092 --list

```

## 3. Configuring the Application

### 3.1 Create Configuration File

Create a copy of the example configuration file:

```bash
# Copy the example configuration
cp config/config.example.json config/config.json
```

### 3.2 Customize Configuration (Optional)

Edit `config/config.json` to customize settings. The most important settings are:

- Redis connection details
- Kafka broker addresses
- API server port
- JWT secret for authentication

```json
{
  "redis": {
    "address": "localhost:6379"
  },
  "kafka": {
    "brokers": "localhost:9092"
  },
  "api": {
    "port": "8080"
  },
  "auth": {
    "jwt_secret": "your_secure_secret_here"
  }
}
```

### 3.3 Configuration Options Reference

The following table describes the key configuration options:

| Section | Option | Description | Default |
|---------|--------|-------------|---------|
| `redis` | `address` | Redis server address | `localhost:6379` |
| `redis` | `password` | Redis password | `""` (empty) |
| `redis` | `db` | Redis database number | `0` |
| `kafka` | `brokers` | Kafka broker addresses | `localhost:9092` |
| `kafka` | `transaction_topic` | Topic for incoming transactions | `transactions` |
| `api` | `port` | API server port | `8080` |
| `api` | `cors_allowed_origins` | CORS allowed origins | `["*"]` |
| `auth` | `jwt_secret` | Secret for JWT token generation | Required |
| `auth` | `jwt_expiration_time` | JWT token expiration time | `24h` |
| `supply` | `min_inflation` | Minimum annual inflation rate | `1.5` |
| `supply` | `max_inflation` | Maximum annual inflation rate | `3.0` |
| `log` | `level` | Log level (debug, info, warn, error) | `info` |

For a complete list of configuration options, see `pkg/config/config.go`.

## 4. Running the Application

### 4.1 Run the Complete System

To run the entire system as a single process:

```bash
# Run with default configuration
go run cmd/stathera/main.go

# Run with custom configuration file
go run cmd/stathera/main.go -config=config/config.json

# Run with specific log level
go run cmd/stathera/main.go -log-level=debug
```

### 4.2 Run Individual Services (Optional)

For development or debugging, you can run individual services:

```bash
# Run the API server
go run cmd/api/main.go

# Run the Order Book service
go run cmd/orderbook/main.go

# Run the Supply Manager service
go run cmd/supply-manager/main.go
```

### 4.3 Verify the Application is Running

Check if the API server is responding:

```bash
# Check the health endpoint
curl http://localhost:8080/health

# Check the API version
curl http://localhost:8080/api/v1
```

## 5. Testing the Application

### 5.1 Run Unit Tests

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/transaction

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...
```

### 5.2 Run Load Tests

The system includes a load testing tool that simulates high transaction volumes:

```bash
# Run load test with default settings
go run cmd/loadtest/main.go -redis=localhost:6379

# Run load test with custom parameters
go run cmd/loadtest/main.go \
  -redis=localhost:6379 \
  -duration=2m \
  -wallets=2000 \
  -concurrency=200 \
  -rate=2000 \
  -balance=20000
```

Load test parameters:
- `duration`: Test duration (default: 1 minute)
- `wallets`: Number of test wallets (default: 1000)
- `concurrency`: Number of concurrent clients (default: 100)
- `rate`: Target transactions per second (default: 1000)
- `balance`: Initial balance for each wallet (default: 10000)

### 5.3 Manual Testing with API Endpoints

You can test the API endpoints using curl or a tool like Postman:

#### Register a new user
```bash
curl -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}'
```

#### Login
```bash
curl -X POST http://localhost:8080/api/v1/login \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","password":"password123"}'
```

#### Create a transaction
```bash
curl -X POST http://localhost:8080/api/v1/transfer \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "receiver_address": "RECEIVER_ADDRESS",
    "amount": 100.0,
    "description": "Test payment",
    "private_key": "YOUR_PRIVATE_KEY"
  }'
```

#### Get transaction history
```bash
curl -X GET http://localhost:8080/api/v1/transactions \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

#### Get order book
```bash
curl -X GET http://localhost:8080/api/v1/orderbook
```

#### Place an order
```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -d '{
    "type": "buy",
    "price": 10.5,
    "amount": 5.0
  }'
```

#### Cancel an order
```bash
curl -X DELETE http://localhost:8080/api/v1/orders/ORDER_ID \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

**Note**: All API endpoints are prefixed with `/api/v1/` in the actual implementation. The API server is configured to use this version prefix as defined in the configuration file.

### 5.4 End-to-End Testing Scenario

Here's a complete end-to-end testing scenario:

1. Start the application:
   ```bash
   go run cmd/stathera/main.go
   ```

2. Register two users:
   ```bash
   # Register user1
   curl -X POST http://localhost:8080/api/v1/register \
     -H "Content-Type: application/json" \
     -d '{"username":"user1","password":"password123"}'
   
   # Register user2
   curl -X POST http://localhost:8080/api/v1/register \
     -H "Content-Type: application/json" \
     -d '{"username":"user2","password":"password123"}'
   ```

3. Login with both users to get JWT tokens:
   ```bash
   # Login user1
   curl -X POST http://localhost:8080/api/v1/login \
     -H "Content-Type: application/json" \
     -d '{"username":"user1","password":"password123"}'
   
   # Login user2
   curl -X POST http://localhost:8080/api/v1/login \
     -H "Content-Type: application/json" \
     -d '{"username":"user2","password":"password123"}'
   ```

4. User1 places a sell order:
   ```bash
   curl -X POST http://localhost:8080/api/v1/orders \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer USER1_JWT_TOKEN" \
     -d '{
       "type": "sell",
       "price": 10.0,
       "amount": 5.0
     }'
   ```

5. User2 places a buy order that matches:
   ```bash
   curl -X POST http://localhost:8080/api/v1/orders \
     -H "Content-Type: application/json" \
     -H "Authorization: Bearer USER2_JWT_TOKEN" \
     -d '{
       "type": "buy",
       "price": 10.0,
       "amount": 5.0
     }'
   ```

6. Check the order book to verify the orders were matched:
   ```bash
   curl -X GET http://localhost:8080/api/v1/orderbook
   ```

7. Check transaction history for both users:
   ```bash
   # User1 transactions
   curl -X GET http://localhost:8080/api/v1/transactions \
     -H "Authorization: Bearer USER1_JWT_TOKEN"
   
   # User2 transactions
   curl -X GET http://localhost:8080/api/v1/transactions \
     -H "Authorization: Bearer USER2_JWT_TOKEN"
   ```

## 6. Implementing New Features

### 6.1 Adding a New Transaction Type

1. Define the new transaction type in `internal/transaction/transaction.go`:
   ```go
   const (
       Payment TransactionType = "PAYMENT"
       // Add your new type
       Staking TransactionType = "STAKING"
   )
   ```

2. Update the `Validate` method to handle the new type:
   ```go
   func (tx *Transaction) Validate() error {
       // Existing validation...
       
       switch tx.Type {
       case Payment:
           // Payment validation
       case Staking:
           // Staking validation
           if tx.Amount < 100 {
               return fmt.Errorf("staking amount must be at least 100")
           }
       }
       
       return nil
   }
   ```

3. Update the transaction processor in `internal/processor/transaction_processor.go` to handle the new type.

### 6.2 Adding a New API Endpoint

1. Define the handler function in `internal/api/server.go`:
   ```go
   func (s *Server) handleStaking(w http.ResponseWriter, r *http.Request) {
       // Implementation
       resp := Response{
           Success: true,
           Data: map[string]interface{}{
               "message": "Staking successful",
           },
       }
       s.renderJSON(w, resp, http.StatusOK)
   }
   ```

2. Add the route to the appropriate router group:
   ```go
   // Add to setupRoutes method
   r.Post("/stake", s.handleStaking)
   ```

### 6.3 Adding a New Service

1. Create a new package in `internal/`:
   ```go
   // internal/notification/service.go
   package notification

   import (
       "context"
       "fmt"
       
       "github.com/cmatc13/stathera/pkg/service"
   )

   // NotificationService sends notifications to users
   type NotificationService struct {
       // Fields
       status service.Status
   }

   // NewNotificationService creates a new notification service
   func NewNotificationService() *NotificationService {
       return &NotificationService{
           status: service.StatusStopped,
       }
   }

   // Name returns the service name
   func (s *NotificationService) Name() string {
       return "notification"
   }

   // Start initializes and starts the service
   func (s *NotificationService) Start(ctx context.Context) error {
       s.status = service.StatusStarting
       // Initialization logic
       s.status = service.StatusRunning
       return nil
   }

   // Stop gracefully shuts down the service
   func (s *NotificationService) Stop(ctx context.Context) error {
       s.status = service.StatusStopping
       // Cleanup logic
       s.status = service.StatusStopped
       return nil
   }

   // Status returns the current service status
   func (s *NotificationService) Status() service.Status {
       return s.status
   }

   // Health performs a health check
   func (s *NotificationService) Health() error {
       if s.status != service.StatusRunning {
           return fmt.Errorf("service not running")
       }
       return nil
   }

   // Dependencies returns a list of services this service depends on
   func (s *NotificationService) Dependencies() []string {
       return []string{"api"}
   }
   ```

2. Register the service in `cmd/stathera/main.go`:
   ```go
   // Initialize and register notification service
   notificationService := notification.NewNotificationService()
   if err := registry.Register(notificationService); err != nil {
       logger.Error("Failed to register notification service", "error", err)
       os.Exit(1)
   }
   ```

## 7. Troubleshooting

### 7.1 Common Issues

#### Redis Connection Errors
```
Error: failed to initialize Redis ledger: dial tcp [::1]:6379: connect: connection refused
```

**Solution:**
- Ensure Redis is running: `docker-compose ps`
- Check Redis logs: `docker-compose logs redis`
- Verify Redis address in configuration

#### Kafka Connection Errors
```
Error: failed to create consumer: kafka: client has run out of available brokers
```

**Solution:**
- Ensure Kafka is running: `docker-compose ps`
- Check Kafka logs: `docker-compose logs kafka`
- Verify Kafka broker address in configuration

#### API Authentication Failures
```
Error: invalid or expired JWT token
```

**Solution:**
- Check that the JWT secret in configuration matches what was used to generate tokens
- Verify token expiration time
- Ensure the token is being sent correctly in the Authorization header

### 7.2 Debugging

To enable more detailed logging:

```bash
# Run with debug log level
go run cmd/stathera/main.go -log-level=debug
```

You can also use Go's built-in debugging tools:

```bash
# Run with race detector
go run -race cmd/stathera/main.go

# Build with debug information
go build -gcflags="all=-N -l" cmd/stathera/main.go
```

### 7.3 Checking Logs

Check the logs for specific services:

```bash
# Check Redis logs
docker-compose logs redis

# Check Kafka logs
docker-compose logs kafka

# Check Zookeeper logs
docker-compose logs zookeeper
```

## 8. Monitoring

### 8.1 Health Checks

The system provides health check endpoints:

```bash
# Check overall system health
curl http://localhost:8080/health

# Check specific service health
curl http://localhost:8080/health/api
curl http://localhost:8080/health/transaction-processor
curl http://localhost:8080/health/orderbook
curl http://localhost:8080/health/supply-manager
```

### 8.2 Metrics

If metrics are enabled in the configuration, you can access Prometheus metrics:

```bash
# Access metrics endpoint
curl http://localhost:9090/metrics
```

### 8.3 Monitoring Redis

You can monitor Redis using the Redis CLI:

```bash
# Connect to Redis
redis-cli

# Check Redis info
INFO

# Monitor Redis commands in real-time
MONITOR

# Check Redis memory usage
INFO memory
```

### 8.4 Monitoring Kafka

You can monitor Kafka using Kafka tools:

```bash
# List Kafka topics
kafka-topics.sh --bootstrap-server localhost:9092 --list

# Describe a topic
kafka-topics.sh --bootstrap-server localhost:9092 --describe --topic transactions

# Check consumer groups
kafka-consumer-groups.sh --bootstrap-server localhost:9092 --list

# Describe a consumer group
kafka-consumer-groups.sh --bootstrap-server localhost:9092 --describe --group transaction_processor_group
```

## Conclusion

You now have a complete guide to setting up, running, and testing the Stathera financial platform in a local environment. This guide covers all aspects of the system, from initial setup to troubleshooting and monitoring.

For more information, refer to the following resources:
- [Architecture Documentation](architecture.md)
- [Developer Guide](developer-guide.md)
- [Error Handling Guide](error-handling-guide.md)
- [Security Checklist](security-checklist.md)
