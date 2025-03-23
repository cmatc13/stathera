# Stathera Developer Guide

## Overview

Stathera is a financial system built in Go that provides transaction processing, order book management, and supply management capabilities. This guide will help you understand the system architecture and how to work with it.

## Getting Started

### Prerequisites

- Go 1.24 or later
- Redis
- Kafka
- Git

### Setting Up Your Environment

1. Install Go 1.24 or later
2. Clone the repository
3. Install dependencies:
   ```
   go mod download
   ```
   or use the provided script:
   ```
   ./scripts/download-deps.sh
   ```

### Running the Application

To run the entire system:

```
go run cmd/stathera/main.go
```

To run individual services:

```
go run cmd/api/main.go
go run cmd/orderbook/main.go
go run cmd/supply-manager/main.go
```

### Configuration

Configuration is managed through environment variables and/or a `.env` file. Key configuration parameters:

- `REDIS_ADDRESS`: Redis server address (default: "localhost:6379")
- `KAFKA_BROKERS`: Comma-separated list of Kafka brokers (default: "localhost:9092")
- `API_PORT`: API server port (default: "8080")
- `JWT_SECRET`: Secret key for JWT token generation
- `SUPPLY_MIN_INFLATION`: Minimum annual inflation rate (default: "1.0")
- `SUPPLY_MAX_INFLATION`: Maximum annual inflation rate (default: "5.0")

For a complete list of configuration options, see `pkg/config/config.go`.

## Architecture

The system follows a service-oriented architecture with the following key components:

1. **Transaction Processor**: Handles financial transactions, validates them, and updates account balances.
2. **Order Book**: Manages buy/sell orders and implements price matching algorithms.
3. **Supply Manager**: Controls currency inflation and adjusts supply based on economic indicators.
4. **API Server**: Provides RESTful endpoints for client interactions.

For more details, see the [Architecture Documentation](architecture.md).

## Unified Service Architecture

The system uses a unified service architecture that allows all components to be run from a single entry point. This is implemented through the service registry pattern.

### Service Interface

All services implement the `Service` interface defined in `pkg/service/service.go`:

```go
// Service defines the interface that all services must implement.
// This interface provides methods for managing the lifecycle of a service,
// including starting, stopping, health checking, and dependency management.
type Service interface {
    // Name returns the service name.
    Name() string
    
    // Start initializes and starts the service.
    // It should be non-blocking and return quickly, with any long-running
    // operations started in separate goroutines.
    Start(ctx context.Context) error
    
    // Stop gracefully shuts down the service.
    // It should ensure all resources are properly released and any
    // ongoing operations are completed or terminated gracefully.
    Stop(ctx context.Context) error
    
    // Status returns the current service status.
    // This can be used to check if the service is running, starting, stopping, etc.
    Status() Status
    
    // Health performs a health check and returns error if unhealthy.
    // This is used by the registry to determine if the service is functioning properly.
    Health() error
    
    // Dependencies returns a list of services this service depends on.
    // The registry uses this information to determine the order in which
    // services should be started and stopped.
    Dependencies() []string
}
```

### Service Registry

The service registry (`pkg/service/registry.go`) manages the lifecycle of all services:

```go
// Registry manages all services and their lifecycle.
// It handles service registration, dependency resolution, and coordinated
// startup and shutdown of services.
type Registry struct {
    services map[string]Service
    mutex    sync.RWMutex
    logger   *log.Logger
}

// Register adds a service to the registry.
// It returns an error if a service with the same name is already registered.
func (r *Registry) Register(service Service) error

// StartAll starts all registered services in dependency order.
// It builds a dependency graph, performs a topological sort to determine
// the correct startup order, and starts each service in that order.
// It waits for each service to become healthy before starting the next one.
func (r *Registry) StartAll(ctx context.Context) error

// StopAll stops all registered services in reverse dependency order.
// It builds a dependency graph, performs a topological sort, reverses the order,
// and stops each service in that order. This ensures that services are stopped
// in the correct order to avoid dependency issues.
func (r *Registry) StopAll(ctx context.Context) error
```

### Main Application

The main application (`cmd/stathera/main.go`) initializes and coordinates all services:

1. Loads configuration
2. Creates the service registry
3. Initializes and registers all services
4. Starts all services
5. Handles graceful shutdown

## Error Handling

The system uses a standardized error handling approach implemented in the `pkg/errors` package. This approach provides:

1. Domain-specific error types
2. Consistent error wrapping
3. Context-rich error messages
4. Error type checking

### Error Types

Each domain has its own set of error types defined in the `pkg/errors` package:

- `transaction.go`: Transaction-related errors
- `orderbook.go`: Order book-related errors
- `storage.go`: Storage-related errors
- `api.go`: API-related errors

### Error Structure

Errors are structured to include:

- Domain: The domain where the error occurred (e.g., "orderbook")
- Operation: The operation that failed (e.g., "PlaceOrder")
- Code: A machine-readable error code (e.g., "ORDERBOOK_INVALID_ORDER")
- Message: A human-readable error message
- Original: The original error that was wrapped

### Error Handling Pattern

```go
// Creating a new error
return errors.OrderBookErrorf(
    errors.OrderBookErrOrderNotFound,
    "order %s not found",
    orderID,
)

// Wrapping an error
return errors.OrderBookWrapWithCode(
    err,
    errors.OpGetOrder,
    errors.OrderBookErrRedisOperation,
    "failed to get order",
)

// Checking error types
if errors.IsOrderBookError(err, errors.OrderBookErrOrderNotFound) {
    // Handle not found error
}
```

For more details, see the [Error Handling Guide](error-handling-guide.md) and [Error Handling Examples](error-handling-examples.md).

## Common Development Tasks

### Adding a New API Endpoint

1. Define the handler function in `internal/api/server.go`
2. Add the route to the appropriate router group in `setupRoutes()`

Example:

```go
// Add to setupRoutes method
r.Get("/new-endpoint", s.handleNewEndpoint)

// Add handler
func (s *Server) handleNewEndpoint(w http.ResponseWriter, r *http.Request) {
    // Implementation
    resp := Response{
        Success: true,
        Data: map[string]interface{}{
            "message": "This is a new endpoint",
        },
    }
    s.renderJSON(w, resp, http.StatusOK)
}
```

### Implementing a New Transaction Type

1. Add the new type to `TransactionType` in `internal/transaction/transaction.go`
2. Update the `ProcessTransaction` method in `internal/storage/redis_ledger.go` to handle the new type
3. Add validation logic to the `Validate` method in `internal/transaction/transaction.go`

Example:

```go
// Add to TransactionType
const (
    Payment        TransactionType = "PAYMENT"
    SupplyIncrease TransactionType = "SUPPLY_INCREASE"
    Staking        TransactionType = "STAKING"  // New transaction type
)

// Update Validate method
func (tx *Transaction) Validate() error {
    // Existing validation logic...
    
    // Validate based on transaction type
    switch tx.Type {
    case Payment:
        // Payment validation
    case SupplyIncrease:
        // Supply increase validation
    case Staking:
        // Staking validation
        if tx.Amount < 100 {
            return fmt.Errorf("staking amount must be at least 100")
        }
    default:
        return fmt.Errorf("invalid transaction type: %s", tx.Type)
    }
    
    return nil
}
```

### Modifying the Order Book Algorithm

The order book implementation is in `internal/orderbook/orderbook.go`. Key methods:

- `PlaceOrder`: Adds a new order to the book
- `CancelOrder`: Removes an order from the book
- `MatchOrders`: Matches buy and sell orders

Example of adding a new order type:

```go
// Add to OrderType
const (
    BidOrder OrderType = "BID"
    AskOrder OrderType = "ASK"
    StopOrder OrderType = "STOP"  // New order type
)

// Update PlaceOrder method
func (ob *RedisOrderBook) PlaceOrder(order *Order) error {
    // Existing logic...
    
    switch order.Type {
    case BidOrder:
        // Handle bid order
    case AskOrder:
        // Handle ask order
    case StopOrder:
        // Handle stop order
        return ob.placeStopOrder(order)
    default:
        return fmt.Errorf("invalid order type: %s", order.Type)
    }
    
    return nil
}

// Add new method for stop orders
func (ob *RedisOrderBook) placeStopOrder(order *Order) error {
    // Implementation
}
```

### Adding a New Service

1. Create a new package in `internal/`
2. Implement the core functionality
3. Create a service wrapper that implements the `Service` interface
4. Register the service in `cmd/stathera/main.go`

Example:

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

Then register the service in `cmd/stathera/main.go`:

```go
// Initialize and register notification service
notificationService := notification.NewNotificationService()
if err := registry.Register(notificationService); err != nil {
    logger.Fatalf("Failed to register notification service: %v", err)
}
```

### Implementing a New Interface

To avoid circular dependencies, you can define interfaces in a separate package. For example, the `pkg/transaction` package defines the `Processor` interface:

```go
// pkg/transaction/processor.go
package transaction

import (
    "github.com/cmatc13/stathera/internal/transaction"
)

// Processor defines the interface for submitting transactions.
// This interface is used by components that need to submit transactions
// without directly depending on the transaction processor implementation.
type Processor interface {
    // SubmitTransaction submits a new transaction to be processed.
    SubmitTransaction(tx *transaction.Transaction) error
}
```

Then implement the interface in your service:

```go
// internal/processor/transaction_processor.go
package processor

// SubmitTransaction submits a new transaction to be processed.
// This method implements the pkg/transaction.Processor interface.
func (tp *TransactionProcessor) SubmitTransaction(tx *transaction.Transaction) error {
    // Implementation
}
```

And use the interface in other services:

```go
// internal/supply/manager.go
package supply

import (
    txproc "github.com/cmatc13/stathera/pkg/transaction"
)

type SupplyManager struct {
    // Fields
    txProcessor txproc.Processor
}

func NewSupplyManager(txProcessor txproc.Processor) *SupplyManager {
    return &SupplyManager{
        txProcessor: txProcessor,
    }
}
```

## Testing

### Unit Testing

Unit tests focus on testing individual components in isolation. They should be fast, reliable, and independent.

Example:

```go
// internal/transaction/transaction_test.go
package transaction

import (
    "testing"
)

func TestNewTransaction(t *testing.T) {
    // Test cases
    tests := []struct {
        name           string
        sender         string
        receiver       string
        amount         float64
        fee            float64
        txType         TransactionType
        nonce          string
        description    string
        expectError    bool
        errorSubstring string
    }{
        {
            name:        "Valid transaction",
            sender:      "sender123",
            receiver:    "receiver456",
            amount:      100.0,
            fee:         1.0,
            txType:      Payment,
            nonce:       "nonce123",
            description: "Test payment",
            expectError: false,
        },
        {
            name:           "Invalid amount",
            sender:         "sender123",
            receiver:       "receiver456",
            amount:         -100.0,
            fee:            1.0,
            txType:         Payment,
            nonce:          "nonce123",
            description:    "Test payment",
            expectError:    true,
            errorSubstring: "amount must be positive",
        },
    }
    
    // Run tests
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tx, err := NewTransaction(
                tt.sender,
                tt.receiver,
                tt.amount,
                tt.fee,
                tt.txType,
                tt.nonce,
                tt.description,
            )
            
            if tt.expectError {
                if err == nil {
                    t.Errorf("Expected error but got nil")
                } else if !strings.Contains(err.Error(), tt.errorSubstring) {
                    t.Errorf("Expected error containing %q but got %q", tt.errorSubstring, err.Error())
                }
            } else {
                if err != nil {
                    t.Errorf("Unexpected error: %v", err)
                }
                if tx == nil {
                    t.Errorf("Expected transaction but got nil")
                } else {
                    // Verify transaction fields
                    if tx.Sender != tt.sender {
                        t.Errorf("Expected sender %q but got %q", tt.sender, tx.Sender)
                    }
                    // Verify other fields...
                }
            }
        })
    }
}
```

### Integration Testing

Integration tests focus on testing the interaction between components. They should verify that components work together correctly.

Example:

```go
// internal/processor/integration_test.go
package processor

import (
    "context"
    "testing"
    
    "github.com/cmatc13/stathera/internal/transaction"
    "github.com/cmatc13/stathera/pkg/config"
)

func TestTransactionProcessing(t *testing.T) {
    // Skip if not running integration tests
    if testing.Short() {
        t.Skip("Skipping integration test in short mode")
    }
    
    // Set up test configuration
    cfg := &config.Config{
        Redis: config.RedisConfig{
            Address: "localhost:6379",
        },
        Kafka: config.KafkaConfig{
            Brokers: "localhost:9092",
        },
    }
    
    // Create processor
    ctx := context.Background()
    processor, err := NewTransactionProcessor(ctx, cfg)
    if err != nil {
        t.Fatalf("Failed to create processor: %v", err)
    }
    
    // Create test transaction
    tx, err := transaction.NewTransaction(
        "sender123",
        "receiver456",
        100.0,
        1.0,
        transaction.Payment,
        "nonce123",
        "Test payment",
    )
    if err != nil {
        t.Fatalf("Failed to create transaction: %v", err)
    }
    
    // Submit transaction
    err = processor.SubmitTransaction(tx)
    if err != nil {
        t.Fatalf("Failed to submit transaction: %v", err)
    }
    
    // Verify transaction was processed
    // This would typically involve checking the database or other state
}
```

### Running Tests

Run all tests:

```
go test ./...
```

Run tests for a specific package:

```
go test ./internal/processor
```

Run tests with verbose output:

```
go test -v ./...
```

Run tests with coverage:

```
go test -cover ./...
```

## Troubleshooting

### Common Issues

1. **Redis Connection Errors**
   - Check if Redis is running
   - Verify the Redis address in configuration
   - Check Redis authentication settings

2. **Kafka Connection Errors**
   - Check if Kafka is running
   - Verify the Kafka brokers in configuration
   - Check Kafka topic configuration

3. **API Authentication Failures**
   - Check the JWT secret configuration
   - Verify token expiration and claims
   - Check that the client is sending the token correctly

### Logging

The system uses the standard Go logger. To increase verbosity, set the `LOG_LEVEL` environment variable:

```
export LOG_LEVEL=debug
```

Log levels:
- `debug`: Detailed debugging information
- `info`: Informational messages
- `warn`: Warning messages
- `error`: Error messages

### Monitoring

Health checks are available at `/health` for each service. Use these to monitor service status.

## Best Practices

1. **Error Handling**: Always wrap errors with context using the `pkg/errors` package.
2. **Configuration**: Add new configuration parameters to `pkg/config/config.go`.
3. **Documentation**: Add godoc comments to all exported functions and types.
4. **Testing**: Write tests for all new functionality.
5. **Dependency Management**: Keep dependencies up to date and minimal.
6. **Code Style**: Follow Go best practices and use `gofmt` to format code.
7. **Interfaces**: Define interfaces for dependencies to make testing easier.
8. **Concurrency**: Use channels and goroutines carefully, and always handle context cancellation.

## API Reference

### Authentication

- `POST /login`: Authenticate a user and get a JWT token
- `POST /register`: Register a new user

### Transactions

- `POST /transfer`: Create a new payment transaction
- `GET /transactions`: Get transaction history

### Order Book

- `GET /orderbook`: Get the current order book
- `POST /orders`: Place a new order
- `DELETE /orders/{id}`: Cancel an order

### System

- `GET /health`: Check system health
- `GET /admin/system/supply`: Get total supply (admin only)
- `GET /admin/system/inflation`: Get inflation rate (admin only)
- `POST /admin/system/adjust-inflation`: Adjust inflation parameters (admin only)

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for your changes
5. Run the tests
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
