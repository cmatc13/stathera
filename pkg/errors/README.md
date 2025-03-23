# Standardized Error Handling

This package provides a standardized approach to error handling in the Stathera project. It defines domain-specific error types, consistent error wrapping, and adds context to errors for better debugging.

## Key Features

1. **Domain-Specific Error Types**: Errors are categorized by domain (e.g., orderbook, transaction, storage, api) with specific error codes.
2. **Error Wrapping**: Consistent approach to wrapping errors with additional context.
3. **Rich Error Context**: Errors include domain, operation, code, message, and additional fields.
4. **Stack Traces**: Optional stack traces for better debugging.
5. **Error Checking**: Helper functions to check error types and codes.

## Usage Examples

### Creating Domain-Specific Errors

```go
// Create a new orderbook error
err := errors.NewOrderBookError(
    errors.OrderBookErrInvalidOrder,
    "Invalid order price",
    nil,
)

// Create a new transaction error with formatted message
err := errors.TransactionErrorf(
    errors.TransactionErrInvalidAmount,
    "Amount %f is invalid", 
    amount,
)
```

### Wrapping Errors

```go
// Wrap an error with domain and operation
if err := redisClient.Set(ctx, key, value, 0).Err(); err != nil {
    return errors.StorageWrap(err, errors.OpSet, "Failed to store order")
}

// Wrap with domain, operation, and code
if err := validateTransaction(tx); err != nil {
    return errors.TransactionWrapWithCode(
        err,
        errors.OpValidateTransaction,
        errors.TransactionErrInvalidAmount,
        "Transaction validation failed",
    )
}
```

### Adding Context to Errors

```go
// Add fields to provide more context
err = errors.WrapWithField(err, "order_id", order.ID)
err = errors.WrapWithField(err, "user_id", order.UserID)
```

### Checking Error Types

```go
// Check if an error is a specific domain error
if errors.IsOrderBookError(err, errors.OrderBookErrOrderNotFound) {
    // Handle not found error
}

// Check if an error is a specific standard error
if errors.Is(err, errors.ErrNotFound) {
    // Handle not found error
}
```

### Using Stack Traces

```go
// Add a stack trace to an error
err = errors.WithStack(err)
```

### Convenience Function

```go
// Create an error with multiple attributes
err := errors.E(
    "Failed to process order",
    errors.OrderBookDomain,
    errors.OpProcessMatch,
    errors.OrderBookErrProcessingFailed,
    originalErr,
    map[string]interface{}{
        "order_id": order.ID,
        "price": order.Price,
    },
)
```

## Error Domains

The package defines several error domains:

- **OrderBook**: Errors related to the orderbook functionality
- **Transaction**: Errors related to transactions
- **Storage**: Errors related to data storage
- **API**: Errors related to the API layer

Each domain has its own set of error codes and operations.

## Best Practices

1. **Use Domain-Specific Errors**: Create errors using the appropriate domain functions.
2. **Add Context**: Always add relevant context to errors, such as the operation that failed.
3. **Include Original Error**: When wrapping errors, always include the original error.
4. **Use Codes Consistently**: Use the predefined error codes consistently throughout the codebase.
5. **Check Error Types**: Use the provided functions to check error types and codes.
6. **Add Stack Traces**: For critical errors, add stack traces to help with debugging.
