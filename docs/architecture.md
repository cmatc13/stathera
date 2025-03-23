# Stathera Architecture

## System Overview

Stathera is a financial platform that provides transaction processing, order book management, and supply management capabilities. The system is designed to be scalable, reliable, and maintainable.

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                        Stathera System                           │
│                                                                 │
│  ┌───────────┐     ┌───────────┐     ┌───────────────────────┐  │
│  │           │     │           │     │                       │  │
│  │  API      │     │  Kafka    │     │  Redis                │  │
│  │  Server   │◄────┤  Broker   │◄────┤  Storage              │  │
│  │           │     │           │     │                       │  │
│  └───────────┘     └───────────┘     └───────────────────────┘  │
│        ▲                                        ▲               │
│        │                                        │               │
│        ▼                                        ▼               │
│  ┌───────────┐     ┌───────────┐     ┌───────────────────────┐  │
│  │           │     │           │     │                       │  │
│  │Transaction│     │ Order     │     │ Supply                │  │
│  │Processor  │◄────┤ Book      │◄────┤ Manager               │  │
│  │           │     │           │     │                       │  │
│  └───────────┘     └───────────┘     └───────────────────────┘  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

## Core Components

### 1. Transaction Processor

The Transaction Processor is responsible for validating and executing financial transactions. It ensures that transactions are properly signed, have sufficient funds, and are recorded in the ledger.

**Key Responsibilities:**
- Transaction validation
- Signature verification
- Balance updates
- Transaction history maintenance

**Implementation:**
- Located in `internal/processor`
- Uses Redis for storage
- Uses Kafka for transaction messaging
- Implements the `pkg/transaction.Processor` interface

**Key Classes:**
- `TransactionProcessor`: Main processor implementation
- `TransactionProcessorService`: Service wrapper for the processor

### 2. Order Book

The Order Book manages buy and sell orders for financial instruments. It implements price matching algorithms and maintains the state of all active orders.

**Key Responsibilities:**
- Order placement and cancellation
- Price matching
- Order execution
- Market data provision

**Implementation:**
- Located in `internal/orderbook`
- Uses Redis for order storage
- Implements price-time priority matching

**Key Classes:**
- `RedisOrderBook`: Main orderbook implementation
- `OrderBookService`: Service wrapper for the orderbook
- `Order`: Represents a buy or sell order

### 3. Supply Manager

The Supply Manager controls the currency supply and inflation rate. It adjusts the supply based on economic indicators and mints new coins through system transactions.

**Key Responsibilities:**
- Inflation rate management
- Supply adjustment
- Reserve management
- Economic indicator monitoring

**Implementation:**
- Located in `internal/supply`
- Uses Redis for state storage
- Implements algorithmic supply control
- Uses the Transaction Processor to create supply increase transactions

**Key Classes:**
- `SupplyManager`: Main supply manager implementation
- `SupplyManagerService`: Service wrapper for the supply manager

### 4. API Server

The API Server provides RESTful endpoints for client interactions. It handles authentication, request validation, and response formatting.

**Key Responsibilities:**
- Client request handling
- Authentication and authorization
- Request validation
- Response formatting

**Implementation:**
- Located in `internal/api`
- Uses Chi router
- Implements JWT authentication

**Key Classes:**
- `Server`: Main API server implementation
- `APIService`: Service wrapper for the API server

## Data Flow

1. **Transaction Flow:**
   - Client submits transaction via API
   - API validates request and forwards to Transaction Processor
   - Transaction Processor validates transaction and updates balances
   - Transaction is recorded in the ledger
   - Confirmation is sent back to client

2. **Order Flow:**
   - Client submits order via API
   - API validates request and forwards to Order Book
   - Order Book adds order to the book
   - Order Book attempts to match the order
   - If matched, transactions are created and sent to Transaction Processor
   - Confirmation is sent back to client

3. **Supply Adjustment Flow:**
   - Supply Manager monitors economic indicators
   - Supply Manager calculates new inflation rate
   - Supply Manager mints new coins via system transaction
   - Transaction Processor processes the system transaction
   - New coins are added to reserve account

## Service Architecture

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
```

The registry ensures that services are started in dependency order and stopped in reverse order. It also provides health checking capabilities.

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

## Deployment Architecture

The system can be deployed in various configurations:

### 1. Monolithic Deployment

All services run in a single process. This is the simplest deployment model and is suitable for development and small-scale deployments.

**Advantages:**
- Simple deployment
- Low operational overhead
- Easy debugging

**Disadvantages:**
- Limited scalability
- Single point of failure
- Resource contention

### 2. Microservices Deployment

Each service runs in its own process. This allows for independent scaling and deployment of services.

**Advantages:**
- Independent scaling
- Improved fault isolation
- Technology flexibility

**Disadvantages:**
- Increased operational complexity
- Network overhead
- Distributed debugging challenges

### 3. Hybrid Deployment

Some services are grouped together while others run independently. This allows for optimizing resource usage while maintaining flexibility.

**Advantages:**
- Balanced approach
- Optimized resource usage
- Selective scaling

**Disadvantages:**
- More complex than monolithic
- Requires careful service grouping
- Potential for uneven scaling

## Security Architecture

### Authentication

- JWT-based authentication
- Token expiration and refresh
- Role-based access control

### Authorization

- Role-based access control
- Resource ownership verification
- Action-based permissions

### Data Protection

- TLS for all communications
- Sensitive data encryption
- Secure key management

## Monitoring and Observability

### Logging

- Structured logging
- Log levels (debug, info, warn, error)
- Request ID tracking

### Metrics

- Service-level metrics
- System-level metrics
- Business metrics

### Health Checks

- Service health endpoints
- Dependency health checks
- System health dashboard

## Interface Contracts

### Transaction Processor Interface

The Transaction Processor implements the `pkg/transaction.Processor` interface:

```go
// Processor defines the interface for submitting transactions.
// This interface is used by components that need to submit transactions
// without directly depending on the transaction processor implementation.
type Processor interface {
    // SubmitTransaction submits a new transaction to be processed.
    SubmitTransaction(tx *transaction.Transaction) error
}
```

This interface is used by the Supply Manager to submit supply increase transactions without creating a circular dependency.

## Future Enhancements

1. **Distributed Transaction Processing**
   - Sharded transaction processing
   - Consensus-based validation

2. **Advanced Order Matching**
   - Support for different order types
   - Improved matching algorithms

3. **Enhanced Supply Management**
   - More sophisticated economic models
   - Automated policy adjustments

4. **Improved API**
   - GraphQL support
   - WebSocket for real-time updates
   - API versioning

5. **Enhanced Security**
   - Multi-factor authentication
   - Advanced fraud detection
   - Audit logging
