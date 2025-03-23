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
type Service interface {
    Name() string
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Status() Status
    Health() error
    Dependencies() []string
}
```

### Service Registry

The service registry (`pkg/service/registry.go`) manages the lifecycle of all services:

```go
type Registry struct {
    services map[string]Service
    logger   *log.Logger
}
```

The registry ensures that services are started in dependency order and stopped in reverse order.

### Main Application

The main application (`cmd/stathera/main.go`) initializes and coordinates all services:

1. Loads configuration
2. Creates the service registry
3. Initializes and registers all services
4. Starts all services
5. Handles graceful shutdown

## Deployment Architecture

The system can be deployed in various configurations:

### 1. Monolithic Deployment

All services run in a single process. This is the simplest deployment model and is suitable for development and small-scale deployments.

### 2. Microservices Deployment

Each service runs in its own process. This allows for independent scaling and deployment of services.

### 3. Hybrid Deployment

Some services are grouped together while others run independently. This allows for optimizing resource usage while maintaining flexibility.

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
