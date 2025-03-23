# Error Handling Guide

This guide explains how to implement the standardized error handling approach in the Stathera project.

## Overview

The Stathera project uses a standardized error handling approach that:

1. Defines domain-specific error types
2. Uses error wrapping consistently
3. Adds context to errors for better debugging

This approach makes it easier to:

- Identify the source of errors
- Understand the context in which errors occurred
- Handle errors appropriately based on their type
- Provide meaningful error messages to users
- Debug issues in production

## Error Package Structure

The error handling approach is implemented in the `pkg/errors` package, which consists of:

- `errors.go`: Core error handling functionality
- `orderbook.go`: OrderBook domain-specific errors
- `transaction.go`: Transaction domain-specific errors
- `storage.go`: Storage domain-specific errors
- `api.go`: API domain-specific errors

## Implementation Steps

To implement the standardized error handling approach in your code:

### 1. Import the errors package

```go
import "github.com/cmatc13/stathera/pkg/errors"
```

### 2. Use domain-specific error types

Instead of using generic errors, use domain-specific error types:

```go
// Before
return fmt.Errorf("failed to connect to Redis: %w", err)

// After
return errors.OrderBookWrapWithCode(
    err,
    errors.OpInitialize,
    errors.OrderBookErrRedisConnection,
    "failed to connect to Redis",
)
```

### 3. Add context to errors

When wrapping errors, add context such as the operation that failed:

```go
// Before
return fmt.Errorf("failed to get order: %w", err)

// After
return errors.OrderBookWrapWithCode(
    err,
    errors.OpGetOrder,
    errors.OrderBookErrRedisOperation,
    "failed to get order",
)
```

### 4. Use error checking functions

When handling errors, use the provided error checking functions:

```go
// Check if it's a specific domain error
if errors.IsOrderBookError(err, errors.OrderBookErrOrderNotFound) {
    // Handle not found error
}

// Check if it's a specific standard error
if errors.Is(err, errors.ErrNotFound) {
    // Handle not found error
}

// Extract domain error details
var domainErr *errors.Error
if errors.As(err, &domainErr) {
    log.Printf("Domain: %s, Operation: %s, Code: %s, Message: %s",
        domainErr.Domain, domainErr.Operation, domainErr.Code, domainErr.Message)
}
```

## Error Handling Patterns

### Function Return Errors

When a function returns an error, wrap it with domain-specific context:

```go
func GetOrder(orderID string) (*Order, error) {
    orderJSON, err := redisClient.Get(ctx, orderPrefix+orderID).Result()
    if err != nil {
        if err == redis.Nil {
            return nil, errors.OrderBookErrorf(
                errors.OrderBookErrOrderNotFound,
                "order %s not found",
                orderID,
            )
        }
        return nil, errors.OrderBookWrapWithCode(
            err,
            errors.OpGetOrder,
            errors.OrderBookErrRedisOperation,
            "failed to get order",
        )
    }
    
    var order Order
    if err := json.Unmarshal([]byte(orderJSON), &order); err != nil {
        return nil, errors.OrderBookWrapWithCode(
            err,
            errors.OpGetOrder,
            errors.OrderBookErrInvalidOrder,
            "failed to unmarshal order",
        )
    }
    
    return &order, nil
}
```

### HTTP Handler Errors

In HTTP handlers, convert domain errors to appropriate HTTP responses:

```go
func handleGetOrder(w http.ResponseWriter, r *http.Request) {
    orderID := chi.URLParam(r, "id")
    
    order, err := GetOrder(orderID)
    if err != nil {
        var domainErr *errors.Error
        if errors.As(err, &domainErr) {
            // Log the detailed error
            log.Printf("Error getting order: %s", err)
            
            // Return an appropriate HTTP response
            if errors.IsOrderBookError(err, errors.OrderBookErrOrderNotFound) {
                renderError(w, "Order not found", http.StatusNotFound)
            } else {
                renderError(w, "Failed to get order", http.StatusInternalServerError)
            }
        } else {
            // Unknown error
            log.Printf("Unexpected error getting order: %v", err)
            renderError(w, "Internal server error", http.StatusInternalServerError)
        }
        return
    }
    
    // Render the order
    renderJSON(w, order, http.StatusOK)
}
```

### Error Logging

When logging errors, include the full error message to capture all context:

```go
if err != nil {
    log.Printf("Error processing order: %s", err)
    // Handle error
}
```

## Best Practices

1. **Be Consistent**: Use the same error handling approach throughout the codebase.
2. **Add Context**: Always add relevant context to errors, such as the operation that failed.
3. **Use Domain-Specific Errors**: Use domain-specific error types and codes.
4. **Check Error Types**: Use the provided error checking functions to handle errors appropriately.
5. **Log Detailed Errors**: Log detailed errors for debugging, but return user-friendly messages to clients.
6. **Add Stack Traces**: For critical errors, add stack traces to help with debugging.
7. **Don't Expose Internal Details**: Don't expose internal error details to clients, especially in production.

## Example Workflow

1. Define domain-specific error types in the appropriate domain file (e.g., `orderbook.go`).
2. Use these error types when returning errors from functions.
3. Wrap errors with additional context when propagating them up the call stack.
4. Check error types when handling errors to provide appropriate responses.
5. Log detailed errors for debugging, but return user-friendly messages to clients.

## Conclusion

By following this standardized error handling approach, we can make our code more maintainable, easier to debug, and provide better error messages to users. It also makes it easier to handle errors appropriately based on their type and context.

For more detailed examples, see the [Error Handling Examples](error-handling-examples.md) document.
