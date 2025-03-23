// Package errors provides a standardized error handling approach for the Stathera project.
package errors

// OrderBook error codes define specific error conditions in the orderbook domain.
const (
	// OrderBookErrInvalidOrder indicates an invalid order format or parameters.
	OrderBookErrInvalidOrder = "ORDERBOOK_INVALID_ORDER"
	// OrderBookErrOrderNotFound indicates an order was not found in the orderbook.
	OrderBookErrOrderNotFound = "ORDERBOOK_ORDER_NOT_FOUND"
	// OrderBookErrUnauthorized indicates an unauthorized operation on an order.
	OrderBookErrUnauthorized = "ORDERBOOK_UNAUTHORIZED"
	// OrderBookErrInvalidOrderStatus indicates an invalid order status transition.
	OrderBookErrInvalidOrderStatus = "ORDERBOOK_INVALID_STATUS"
	// OrderBookErrRedisConnection indicates a Redis connection error in the orderbook service.
	OrderBookErrRedisConnection = "ORDERBOOK_REDIS_CONNECTION"
	// OrderBookErrRedisOperation indicates a Redis operation error in the orderbook service.
	OrderBookErrRedisOperation = "ORDERBOOK_REDIS_OPERATION"
	// OrderBookErrMatchingFailed indicates a matching operation failed in the orderbook.
	OrderBookErrMatchingFailed = "ORDERBOOK_MATCHING_FAILED"
	// OrderBookErrProcessingFailed indicates a processing operation failed in the orderbook.
	OrderBookErrProcessingFailed = "ORDERBOOK_PROCESSING_FAILED"
)

// OrderBookDomain is the domain name for orderbook errors.
const OrderBookDomain = "orderbook"

// OrderBook operations define the specific operations that can fail in the orderbook domain.
const (
	// OpPlaceOrder is the operation for placing a new order.
	OpPlaceOrder = "PlaceOrder"
	// OpCancelOrder is the operation for canceling an existing order.
	OpCancelOrder = "CancelOrder"
	// OpMatchOrder is the operation for matching orders.
	OpMatchOrder = "MatchOrder"
	// OpProcessMatch is the operation for processing a match between orders.
	OpProcessMatch = "ProcessMatch"
	// OpGetOrderBook is the operation for retrieving the orderbook state.
	OpGetOrderBook = "GetOrderBook"
	// OpInitialize is the operation for initializing the orderbook service.
	OpInitialize = "Initialize"
	// OpClose is the operation for closing the orderbook service.
	OpClose = "Close"
	// OpGetOrder is the operation for retrieving a specific order.
	OpGetOrder = "GetOrder"
	// OpUpdateOrder is the operation for updating an existing order.
	OpUpdateOrder = "UpdateOrder"
	// OpGetUserOrders is the operation for retrieving all orders for a user.
	OpGetUserOrders = "GetUserOrders"
)

// NewOrderBookError creates a new orderbook error with the specified code, message, and underlying error.
// This function is used to create domain-specific errors in the orderbook domain.
func NewOrderBookError(code string, message string, err error) error {
	return &Error{
		Domain:   OrderBookDomain,
		Code:     code,
		Message:  message,
		Original: err,
	}
}

// OrderBookErrorf creates a new orderbook error with a formatted message.
// This function is used to create domain-specific errors in the orderbook domain
// with a formatted message string.
func OrderBookErrorf(code string, format string, args ...interface{}) error {
	return &Error{
		Domain:  OrderBookDomain,
		Code:    code,
		Message: Sprintf(format, args...),
	}
}

// OrderBookWrap wraps an error with orderbook domain context.
// This function is used to add orderbook domain context to an existing error.
func OrderBookWrap(err error, operation string, message string) error {
	if err == nil {
		return nil
	}

	return &Error{
		Domain:    OrderBookDomain,
		Operation: operation,
		Message:   message,
		Original:  err,
	}
}

// OrderBookWrapWithCode wraps an error with orderbook domain context and a specific error code.
// This function is used to add orderbook domain context and a specific error code to an existing error.
func OrderBookWrapWithCode(err error, operation string, code string, message string) error {
	if err == nil {
		return nil
	}

	return &Error{
		Domain:    OrderBookDomain,
		Operation: operation,
		Code:      code,
		Message:   message,
		Original:  err,
	}
}

// IsOrderBookError checks if an error is an orderbook error with the given code.
// This function is used to check if an error is a specific type of orderbook error.
func IsOrderBookError(err error, code string) bool {
	var domainErr *Error
	if As(err, &domainErr) {
		return domainErr.Domain == OrderBookDomain && domainErr.Code == code
	}
	return false
}
