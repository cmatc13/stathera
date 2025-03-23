// pkg/errors/transaction.go
package errors

// Transaction error codes
const (
	// TransactionErrInvalidAmount indicates an invalid transaction amount
	TransactionErrInvalidAmount = "TRANSACTION_INVALID_AMOUNT"
	// TransactionErrInvalidSender indicates an invalid sender
	TransactionErrInvalidSender = "TRANSACTION_INVALID_SENDER"
	// TransactionErrInvalidReceiver indicates an invalid receiver
	TransactionErrInvalidReceiver = "TRANSACTION_INVALID_RECEIVER"
	// TransactionErrInsufficientFunds indicates insufficient funds
	TransactionErrInsufficientFunds = "TRANSACTION_INSUFFICIENT_FUNDS"
	// TransactionErrInvalidSignature indicates an invalid signature
	TransactionErrInvalidSignature = "TRANSACTION_INVALID_SIGNATURE"
	// TransactionErrInvalidHash indicates an invalid hash
	TransactionErrInvalidHash = "TRANSACTION_INVALID_HASH"
	// TransactionErrDuplicate indicates a duplicate transaction
	TransactionErrDuplicate = "TRANSACTION_DUPLICATE"
	// TransactionErrProcessingFailed indicates a processing failure
	TransactionErrProcessingFailed = "TRANSACTION_PROCESSING_FAILED"
	// TransactionErrInvalidType indicates an invalid transaction type
	TransactionErrInvalidType = "TRANSACTION_INVALID_TYPE"
	// TransactionErrInvalidStatus indicates an invalid transaction status
	TransactionErrInvalidStatus = "TRANSACTION_INVALID_STATUS"
	// TransactionErrKafkaConnection indicates a Kafka connection error
	TransactionErrKafkaConnection = "TRANSACTION_KAFKA_CONNECTION"
	// TransactionErrKafkaOperation indicates a Kafka operation error
	TransactionErrKafkaOperation = "TRANSACTION_KAFKA_OPERATION"
)

// Transaction domain name
const TransactionDomain = "transaction"

// Transaction operations
const (
	OpCreateTransaction    = "CreateTransaction"
	OpValidateTransaction  = "ValidateTransaction"
	OpSubmitTransaction    = "SubmitTransaction"
	OpProcessTransaction   = "ProcessTransaction"
	OpSignTransaction      = "SignTransaction"
	OpVerifyTransaction    = "VerifyTransaction"
	OpGetTransaction       = "GetTransaction"
	OpGetUserTransactions  = "GetUserTransactions"
	OpCalculateHash        = "CalculateHash"
	OpSerializeTransaction = "SerializeTransaction"
)

// NewTransactionError creates a new transaction error
func NewTransactionError(code string, message string, err error) error {
	return &Error{
		Domain:   TransactionDomain,
		Code:     code,
		Message:  message,
		Original: err,
	}
}

// TransactionErrorf creates a new transaction error with formatted message
func TransactionErrorf(code string, format string, args ...interface{}) error {
	return &Error{
		Domain:  TransactionDomain,
		Code:    code,
		Message: Sprintf(format, args...),
	}
}

// TransactionWrap wraps an error with transaction domain
func TransactionWrap(err error, operation string, message string) error {
	if err == nil {
		return nil
	}

	return &Error{
		Domain:    TransactionDomain,
		Operation: operation,
		Message:   message,
		Original:  err,
	}
}

// TransactionWrapWithCode wraps an error with transaction domain and code
func TransactionWrapWithCode(err error, operation string, code string, message string) error {
	if err == nil {
		return nil
	}

	return &Error{
		Domain:    TransactionDomain,
		Operation: operation,
		Code:      code,
		Message:   message,
		Original:  err,
	}
}

// IsTransactionError checks if an error is a transaction error with the given code
func IsTransactionError(err error, code string) bool {
	var domainErr *Error
	if As(err, &domainErr) {
		return domainErr.Domain == TransactionDomain && domainErr.Code == code
	}
	return false
}
