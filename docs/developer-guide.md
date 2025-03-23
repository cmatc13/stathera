# Stathera Developer Guide

## Overview

Stathera is a financial system built in Go that provides transaction processing, order book management, and supply management capabilities. This guide will help you understand the system architecture and how to work with it.

## Architecture

The system follows a service-oriented architecture with the following key components:

1. **Transaction Processor**: Handles financial transactions, validates them, and updates account balances.
2. **Order Book**: Manages buy/sell orders and implements price matching algorithms.
3. **Supply Manager**: Controls currency inflation and adjusts supply based on economic indicators.
4. **API Server**: Provides RESTful endpoints for client interactions.

## Unified Service Architecture

The system uses a unified service architecture that allows all components to be run from a single entry point. This is implemented through the service registry pattern.

### Service Interface

All services implement the `Service` interface defined in `pkg/service/service.go`:

```go
type Service interface {
    // Name returns the service name
    Name() string
    
    // Start initializes and starts the service
    Start(ctx context.Context) error
    
    // Stop gracefully shuts down the service
    Stop(ctx context.Context) error
    
    // Status returns the current service status
    Status() Status
    
    // Health performs a health check
    Health() error
    
    // Dependencies returns a list of services this service depends on
    Dependencies() []string
}
```

### Service Registry

The service registry (`pkg/service/registry.go`) manages the lifecycle of all services:

```go
// Registry manages multiple services
type Registry struct {
    services map[string]Service
    logger   *log.Logger
}

// Register adds a service to the registry
func (r *Registry) Register(service Service) error

// StartAll starts all registered services in dependency order
func (r *Registry) StartAll(ctx context.Context) error

// StopAll stops all registered services in reverse dependency order
func (r *Registry) StopAll(ctx context.Context) error
```

### Main Application

The main application (`cmd/stathera/main.go`) initializes and coordinates all services:

1. Loads configuration
2. Creates the service registry
3. Initializes and registers all services
4. Starts all services
5. Handles graceful shutdown

## Development Workflow

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

### Adding a New Service

1. Create a new package in `internal/`
2. Implement the core functionality
3. Create a service wrapper that implements the `Service` interface
4. Register the service in `cmd/stathera/main.go`

Example:

```go
// Create the service
newService := mypackage.NewMyService(cfg)

// Register it with the registry
if err := registry.Register(newService); err != nil {
    log.Fatalf("Failed to register my service: %v", err)
}
```

### Testing

Run all tests:

```
go test ./...
```

Run tests for a specific package:

```
go test ./internal/processor
```

## Common Tasks

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
}
```

### Implementing a New Transaction Type

1. Add the new type to `TransactionType` in `internal/transaction/transaction.go`
2. Update the `ProcessTransaction` method in `internal/storage/redis_ledger.go` to handle the new type
3. Add validation logic to the `Validate` method in `internal/transaction/transaction.go`

### Modifying the Order Book Algorithm

The order book implementation is in `internal/orderbook/orderbook.go`. Key methods:

- `PlaceOrder`: Adds a new order to the book
- `CancelOrder`: Removes an order from the book
- `MatchOrders`: Matches buy and sell orders

## Troubleshooting

### Common Issues

1. **Redis Connection Errors**
   - Check if Redis is running
   - Verify the Redis address in configuration

2. **Kafka Connection Errors**
   - Check if Kafka is running
   - Verify the Kafka brokers in configuration

3. **API Authentication Failures**
   - Check the JWT secret configuration
   - Verify token expiration and claims

### Logging

The system uses the standard Go logger. To increase verbosity, set the `LOG_LEVEL` environment variable:

```
export LOG_LEVEL=debug
```

### Monitoring

Health checks are available at `/health` for each service. Use these to monitor service status.

## Best Practices

1. **Error Handling**: Always wrap errors with context using `fmt.Errorf("context: %w", err)`
2. **Configuration**: Add new configuration parameters to `pkg/config/config.go`
3. **Documentation**: Add godoc comments to all exported functions and types
4. **Testing**: Write tests for all new functionality
5. **Dependency Management**: Keep dependencies up to date and minimal
