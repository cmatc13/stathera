// pkg/errors/orderbook.go
package errors

// OrderBook error codes
const (
	// OrderBookErrInvalidOrder indicates an invalid order
	OrderBookErrInvalidOrder = "ORDERBOOK_INVALID_ORDER"
	// OrderBookErrOrderNotFound indicates an order was not found
	OrderBookErrOrderNotFound = "ORDERBOOK_ORDER_NOT_FOUND"
	// OrderBookErrUnauthorized indicates an unauthorized operation on an order
	OrderBookErrUnauthorized = "ORDERBOOK_UNAUTHORIZED"
	// OrderBookErrInvalidOrderStatus indicates an invalid order status
	OrderBookErrInvalidOrderStatus = "ORDERBOOK_INVALID_STATUS"
	// OrderBookErrRedisConnection indicates a Redis connection error
	OrderBookErrRedisConnection = "ORDERBOOK_REDIS_CONNECTION"
	// OrderBookErrRedisOperation indicates a Redis operation error
	OrderBookErrRedisOperation = "ORDERBOOK_REDIS_OPERATION"
	// OrderBookErrMatchingFailed indicates a matching operation failed
	OrderBookErrMatchingFailed = "ORDERBOOK_MATCHING_FAILED"
	// OrderBookErrProcessingFailed indicates a processing operation failed
	OrderBookErrProcessingFailed = "ORDERBOOK_PROCESSING_FAILED"
)

// OrderBook domain name
const OrderBookDomain = "orderbook"

// OrderBook operations
const (
	OpPlaceOrder    = "PlaceOrder"
	OpCancelOrder   = "CancelOrder"
	OpMatchOrder    = "MatchOrder"
	OpProcessMatch  = "ProcessMatch"
	OpGetOrderBook  = "GetOrderBook"
	OpInitialize    = "Initialize"
	OpClose         = "Close"
	OpGetOrder      = "GetOrder"
	OpUpdateOrder   = "UpdateOrder"
	OpGetUserOrders = "GetUserOrders"
)

// NewOrderBookError creates a new orderbook error
func NewOrderBookError(code string, message string, err error) error {
	return &Error{
		Domain:   OrderBookDomain,
		Code:     code,
		Message:  message,
		Original: err,
	}
}

// OrderBookErrorf creates a new orderbook error with formatted message
func OrderBookErrorf(code string, format string, args ...interface{}) error {
	return &Error{
		Domain:  OrderBookDomain,
		Code:    code,
		Message: Sprintf(format, args...),
	}
}

// OrderBookWrap wraps an error with orderbook domain
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

// OrderBookWrapWithCode wraps an error with orderbook domain and code
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

// IsOrderBookError checks if an error is an orderbook error with the given code
func IsOrderBookError(err error, code string) bool {
	var domainErr *Error
	if As(err, &domainErr) {
		return domainErr.Domain == OrderBookDomain && domainErr.Code == code
	}
	return false
}
