# Error Handling Examples

This document provides examples of how to use the standardized error handling approach in the Stathera project.

## OrderBook Examples

Here's how to update the `NewRedisOrderBook` function to use the new error handling approach:

```go
// NewRedisOrderBook creates a new Redis-backed order book
func NewRedisOrderBook(redisAddr string) (*RedisOrderBook, error) {
    client := redis.NewClient(&redis.Options{
        Addr: redisAddr,
        DB:   0,
    })

    ctx := context.Background()

    // Test connection
    if _, err := client.Ping(ctx).Result(); err != nil {
        return nil, errors.OrderBookWrapWithCode(
            err,
            errors.OpInitialize,
            errors.OrderBookErrRedisConnection,
            "failed to connect to Redis",
        )
    }

    return &RedisOrderBook{
        client: client,
        ctx:    ctx,
    }, nil
}
```

Here's how to update the `PlaceOrder` function:

```go
// PlaceOrder adds a new order to the order book
func (rob *RedisOrderBook) PlaceOrder(order *Order) error {
    rob.mu.Lock()
    defer rob.mu.Unlock()

    // Store the order
    orderJSON, err := json.Marshal(order)
    if err != nil {
        return errors.OrderBookWrapWithCode(
            err,
            errors.OpPlaceOrder,
            errors.OrderBookErrInvalidOrder,
            "failed to marshal order",
        )
    }

    // Store the order details
    err = rob.client.Set(rob.ctx, orderPrefix+order.ID, orderJSON, 0).Err()
    if err != nil {
        return errors.OrderBookWrapWithCode(
            err,
            errors.OpPlaceOrder,
            errors.OrderBookErrRedisOperation,
            "failed to store order",
        )
    }

    // Add to the appropriate sorted set
    var key string
    var score float64

    if order.Type == BidOrder {
        key = bidOrdersKey
        score = -order.Price // Negative for descending order (highest bids first)
    } else {
        key = askOrdersKey
        score = order.Price // Ascending order (lowest asks first)
    }

    err = rob.client.ZAdd(rob.ctx, key, &redis.Z{
        Score:  score,
        Member: order.ID,
    }).Err()
    if err != nil {
        return errors.OrderBookWrapWithCode(
            err,
            errors.OpPlaceOrder,
            errors.OrderBookErrRedisOperation,
            "failed to add order to sorted set",
        )
    }

    // Add to user's orders
    err = rob.client.ZAdd(rob.ctx, userOrdersPrefix+order.UserID, &redis.Z{
        Score:  float64(order.CreatedAt),
        Member: order.ID,
    }).Err()
    if err != nil {
        return errors.OrderBookWrapWithCode(
            err,
            errors.OpPlaceOrder,
            errors.OrderBookErrRedisOperation,
            "failed to add order to user's orders",
        )
    }

    // Try to match the order
    matches, err := rob.matchOrder(order)
    if err != nil {
        return errors.OrderBookWrapWithCode(
            err,
            errors.OpMatchOrder,
            errors.OrderBookErrMatchingFailed,
            "failed to match order",
        )
    }

    // Process matches
    for _, match := range matches {
        err = rob.processMatch(match)
        if err != nil {
            return errors.OrderBookWrapWithCode(
                err,
                errors.OpProcessMatch,
                errors.OrderBookErrProcessingFailed,
                "failed to process match",
            )
        }
    }

    return nil
}
```

Here's how to update the `CancelOrder` function:

```go
// CancelOrder cancels an open order
func (rob *RedisOrderBook) CancelOrder(orderID, userID string) error {
    rob.mu.Lock()
    defer rob.mu.Unlock()

    // Get order details
    orderJSON, err := rob.client.Get(rob.ctx, orderPrefix+orderID).Result()
    if err != nil {
        if err == redis.Nil {
            return errors.OrderBookErrorf(
                errors.OrderBookErrOrderNotFound,
                "order %s not found",
                orderID,
            )
        }
        return errors.OrderBookWrapWithCode(
            err,
            errors.OpCancelOrder,
            errors.OrderBookErrRedisOperation,
            "failed to get order",
        )
    }

    var order Order
    if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
        return errors.OrderBookWrapWithCode(
            err,
            errors.OpCancelOrder,
            errors.OrderBookErrInvalidOrder,
            "failed to unmarshal order",
        )
    }

    // Verify ownership
    if order.UserID != userID {
        return errors.OrderBookErrorf(
            errors.OrderBookErrUnauthorized,
            "unauthorized: order %s does not belong to user %s",
            orderID,
            userID,
        )
    }

    // Check if order can be cancelled
    if order.Status != Open && order.Status != PartiallyFilled {
        return errors.OrderBookErrorf(
            errors.OrderBookErrInvalidOrderStatus,
            "cannot cancel order with status %s",
            order.Status,
        )
    }

    // Update order status
    order.Status = Cancelled
    order.UpdatedAt = time.Now().Unix()

    // Store updated order
    updatedOrderJSON, err := json.Marshal(order)
    if err != nil {
        return errors.OrderBookWrapWithCode(
            err,
            errors.OpCancelOrder,
            errors.OrderBookErrInvalidOrder,
            "failed to marshal updated order",
        )
    }

    err = rob.client.Set(rob.ctx, orderPrefix+order.ID, updatedOrderJSON, 0).Err()
    if err != nil {
        return errors.OrderBookWrapWithCode(
            err,
            errors.OpCancelOrder,
            errors.OrderBookErrRedisOperation,
            "failed to store updated order",
        )
    }

    // Remove order from order book
    var key string
    if order.Type == BidOrder {
        key = bidOrdersKey
    } else {
        key = askOrdersKey
    }

    err = rob.client.ZRem(rob.ctx, key, order.ID).Err()
    if err != nil {
        return errors.OrderBookWrapWithCode(
            err,
            errors.OpCancelOrder,
            errors.OrderBookErrRedisOperation,
            "failed to remove order from order book",
        )
    }

    return nil
}
```

## Transaction Examples

Here's how to update the `Validate` function in the transaction package:

```go
// Validate checks if the transaction is valid
func (tx *Transaction) Validate() error {
    // Basic validation
    if tx.Amount <= 0 {
        return errors.TransactionErrorf(
            errors.TransactionErrInvalidAmount,
            "transaction amount must be positive",
        )
    }

    if tx.Sender == tx.Receiver && tx.Type == Payment {
        return errors.TransactionErrorf(
            errors.TransactionErrInvalidSender,
            "sender and receiver cannot be the same for payment transactions",
        )
    }

    // Validate hash
    calculatedHash, err := tx.CalculateHash()
    if err != nil {
        return errors.TransactionWrapWithCode(
            err,
            errors.OpCalculateHash,
            errors.TransactionErrInvalidHash,
            "failed to calculate transaction hash",
        )
    }

    if calculatedHash != tx.Hash {
        return errors.TransactionErrorf(
            errors.TransactionErrInvalidHash,
            "transaction hash is invalid",
        )
    }

    return nil
}
```

## API Examples

Here's how to update the `renderError` function in the API server:

```go
// renderError renders an error response
func (s *Server) renderError(w http.ResponseWriter, message string, status int) {
    resp := Response{
        Success: false,
        Error:   message,
    }

    s.renderJSON(w, resp, status)
}

// renderErrorWithCode renders an error response with a specific error code
func (s *Server) renderErrorWithCode(w http.ResponseWriter, err error) {
    var status int
    var message string

    // Check if it's an API error
    if errors.IsAPIError(err, errors.APIErrBadRequest) {
        status = http.StatusBadRequest
        message = err.Error()
    } else if errors.IsAPIError(err, errors.APIErrUnauthorized) {
        status = http.StatusUnauthorized
        message = err.Error()
    } else if errors.IsAPIError(err, errors.APIErrForbidden) {
        status = http.StatusForbidden
        message = err.Error()
    } else if errors.IsAPIError(err, errors.APIErrNotFound) {
        status = http.StatusNotFound
        message = err.Error()
    } else if errors.IsAPIError(err, errors.APIErrMethodNotAllowed) {
        status = http.StatusMethodNotAllowed
        message = err.Error()
    } else if errors.IsAPIError(err, errors.APIErrConflict) {
        status = http.StatusConflict
        message = err.Error()
    } else if errors.IsAPIError(err, errors.APIErrRateLimitExceeded) {
        status = http.StatusTooManyRequests
        message = err.Error()
    } else if errors.IsAPIError(err, errors.APIErrServiceUnavailable) {
        status = http.StatusServiceUnavailable
        message = err.Error()
    } else {
        // Default to internal server error
        status = http.StatusInternalServerError
        message = "Internal server error"
    }

    resp := Response{
        Success: false,
        Error:   message,
    }

    s.renderJSON(w, resp, status)
}
```

## Storage Examples

Here's how to update the `GetBalance` function in the storage package:

```go
// GetBalance gets a user's balance
func (rl *RedisLedger) GetBalance(address string) (float64, error) {
    balance, err := rl.Client.Get(rl.ctx, balancePrefix+address).Float64()
    if err != nil {
        if err == redis.Nil {
            // If the key doesn't exist, return 0 balance
            return 0, nil
        }
        return 0, errors.StorageWrapWithCode(
            err,
            errors.OpGet,
            errors.StorageErrRead,
            "failed to get balance",
        )
    }
    return balance, nil
}
```

## Error Handling in Main Functions

Here's how to handle errors in a main function:

```go
func main() {
    // Initialize the orderbook
    orderbook, err := orderbook.NewRedisOrderBook(config.Redis.Address)
    if err != nil {
        var domainErr *errors.Error
        if errors.As(err, &domainErr) {
            log.Fatalf("Failed to initialize orderbook: %s", err)
        } else {
            log.Fatalf("Unexpected error initializing orderbook: %v", err)
        }
    }

    // Use the orderbook
    // ...
}
```

## Error Handling in HTTP Handlers

Here's how to handle errors in an HTTP handler:

```go
func (s *Server) handlePlaceOrder(w http.ResponseWriter, r *http.Request) {
    // Parse request
    var req struct {
        Type   string  `json:"type"`
        Price  float64 `json:"price"`
        Amount float64 `json:"amount"`
    }

    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        s.renderError(w, "Invalid request", http.StatusBadRequest)
        return
    }

    // Validate input
    if req.Price <= 0 || req.Amount <= 0 {
        err := errors.APIErrorf(
            errors.APIErrValidation,
            "Price and amount must be positive",
        )
        s.renderErrorWithCode(w, err)
        return
    }

    // Get user from JWT token
    _, claims, err := jwtauth.FromContext(r.Context())
    if err != nil {
        err = errors.APIWrapWithCode(
            err,
            errors.OpAuthenticate,
            errors.APIErrUnauthorized,
            "Authentication error",
        )
        s.renderErrorWithCode(w, err)
        return
    }

    userID, ok := claims["user_id"].(string)
    if !ok {
        err = errors.APIErrorf(
            errors.APIErrUnauthorized,
            "Invalid token claims",
        )
        s.renderErrorWithCode(w, err)
        return
    }

    // Determine order type
    var orderType orderbook.OrderType
    if req.Type == "buy" {
        orderType = orderbook.BidOrder
    } else if req.Type == "sell" {
        orderType = orderbook.AskOrder
    } else {
        err = errors.APIErrorf(
            errors.APIErrValidation,
            "Invalid order type",
        )
        s.renderErrorWithCode(w, err)
        return
    }

    // Create order
    order := orderbook.NewOrder(userID, orderType, req.Price, req.Amount)

    // Place order
    err = s.orderbook.PlaceOrder(order)
    if err != nil {
        // Check if it's a domain-specific error
        var domainErr *errors.Error
        if errors.As(err, &domainErr) {
            // Log the detailed error
            log.Printf("Order placement error: %s", err)
            
            // Return a user-friendly error
            s.renderError(w, "Failed to place order", http.StatusInternalServerError)
        } else {
            // Unknown error
            log.Printf("Unexpected error placing order: %v", err)
            s.renderError(w, "Internal server error", http.StatusInternalServerError)
        }
        return
    }

    resp := Response{
        Success: true,
        Message: "Order placed successfully",
        Data: map[string]interface{}{
            "order_id":  order.ID,
            "type":      order.Type,
            "price":     order.Price,
            "amount":    order.Amount,
            "timestamp": order.CreatedAt,
        },
    }

    s.renderJSON(w, resp, http.StatusOK)
}
