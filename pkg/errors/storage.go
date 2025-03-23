// pkg/errors/storage.go
package errors

// Storage error codes
const (
	// StorageErrConnection indicates a connection error
	StorageErrConnection = "STORAGE_CONNECTION"
	// StorageErrRead indicates a read error
	StorageErrRead = "STORAGE_READ"
	// StorageErrWrite indicates a write error
	StorageErrWrite = "STORAGE_WRITE"
	// StorageErrDelete indicates a delete error
	StorageErrDelete = "STORAGE_DELETE"
	// StorageErrNotFound indicates a resource was not found
	StorageErrNotFound = "STORAGE_NOT_FOUND"
	// StorageErrAlreadyExists indicates a resource already exists
	StorageErrAlreadyExists = "STORAGE_ALREADY_EXISTS"
	// StorageErrInvalidKey indicates an invalid key
	StorageErrInvalidKey = "STORAGE_INVALID_KEY"
	// StorageErrInvalidValue indicates an invalid value
	StorageErrInvalidValue = "STORAGE_INVALID_VALUE"
	// StorageErrSerialization indicates a serialization error
	StorageErrSerialization = "STORAGE_SERIALIZATION"
	// StorageErrDeserialization indicates a deserialization error
	StorageErrDeserialization = "STORAGE_DESERIALIZATION"
	// StorageErrTransaction indicates a transaction error
	StorageErrTransaction = "STORAGE_TRANSACTION"
)

// Storage domain name
const StorageDomain = "storage"

// Storage operations
const (
	OpConnect           = "Connect"
	OpDisconnect        = "Disconnect"
	OpGet               = "Get"
	OpSet               = "Set"
	OpDelete            = "Delete"
	OpList              = "List"
	OpIncrement         = "Increment"
	OpDecrement         = "Decrement"
	OpBeginTransaction  = "BeginTransaction"
	OpCommitTransaction = "CommitTransaction"
	OpRollback          = "Rollback"
	OpSerialize         = "Serialize"
	OpDeserialize       = "Deserialize"
)

// NewStorageError creates a new storage error
func NewStorageError(code string, message string, err error) error {
	return &Error{
		Domain:   StorageDomain,
		Code:     code,
		Message:  message,
		Original: err,
	}
}

// StorageErrorf creates a new storage error with formatted message
func StorageErrorf(code string, format string, args ...interface{}) error {
	return &Error{
		Domain:  StorageDomain,
		Code:    code,
		Message: Sprintf(format, args...),
	}
}

// StorageWrap wraps an error with storage domain
func StorageWrap(err error, operation string, message string) error {
	if err == nil {
		return nil
	}

	return &Error{
		Domain:    StorageDomain,
		Operation: operation,
		Message:   message,
		Original:  err,
	}
}

// StorageWrapWithCode wraps an error with storage domain and code
func StorageWrapWithCode(err error, operation string, code string, message string) error {
	if err == nil {
		return nil
	}

	return &Error{
		Domain:    StorageDomain,
		Operation: operation,
		Code:      code,
		Message:   message,
		Original:  err,
	}
}

// IsStorageError checks if an error is a storage error with the given code
func IsStorageError(err error, code string) bool {
	var domainErr *Error
	if As(err, &domainErr) {
		return domainErr.Domain == StorageDomain && domainErr.Code == code
	}
	return false
}
