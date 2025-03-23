// pkg/errors/errors.go
package errors

import (
	"errors"
	"fmt"
	"runtime"
	"strings"
)

// Sprintf is a convenience function for fmt.Sprintf
func Sprintf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

// Standard errors provides a way to check error types
var (
	// Sentinel errors
	ErrNotFound      = errors.New("resource not found")
	ErrAlreadyExists = errors.New("resource already exists")
	ErrInvalidInput  = errors.New("invalid input")
	ErrUnauthorized  = errors.New("unauthorized access")
	ErrForbidden     = errors.New("forbidden action")
	ErrInternal      = errors.New("internal error")
	ErrUnavailable   = errors.New("service unavailable")
	ErrTimeout       = errors.New("operation timed out")
)

// Unwrap provides compatibility with the standard errors package
func Unwrap(err error) error {
	return errors.Unwrap(err)
}

// Is provides compatibility with the standard errors package
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As provides compatibility with the standard errors package
func As(err error, target interface{}) bool {
	return errors.As(err, target)
}

// New creates a new error with the given message
func New(message string) error {
	return errors.New(message)
}

// Error represents a domain error with additional context
type Error struct {
	// Original is the original error
	Original error
	// Domain is the domain of the error (e.g., "orderbook", "transaction", "storage")
	Domain string
	// Code is a machine-readable error code
	Code string
	// Message is a human-readable error message
	Message string
	// Operation is the operation that failed (e.g., "PlaceOrder", "ProcessTransaction")
	Operation string
	// Fields contains additional context about the error
	Fields map[string]interface{}
	// Stack contains the stack trace
	Stack string
}

// Error implements the error interface
func (e *Error) Error() string {
	var sb strings.Builder

	// Format: [Domain.Operation] Code: Message: Original
	sb.WriteString("[")
	if e.Domain != "" {
		sb.WriteString(e.Domain)
		if e.Operation != "" {
			sb.WriteString(".")
			sb.WriteString(e.Operation)
		}
	} else if e.Operation != "" {
		sb.WriteString(e.Operation)
	}
	sb.WriteString("] ")

	if e.Code != "" {
		sb.WriteString("Code=")
		sb.WriteString(e.Code)
		sb.WriteString(": ")
	}

	if e.Message != "" {
		sb.WriteString(e.Message)
	}

	if e.Original != nil {
		if e.Message != "" {
			sb.WriteString(": ")
		}
		sb.WriteString(e.Original.Error())
	}

	return sb.String()
}

// Unwrap implements the errors.Unwrapper interface
func (e *Error) Unwrap() error {
	return e.Original
}

// WithStack adds a stack trace to the error
func WithStack(err error) error {
	if err == nil {
		return nil
	}

	// Check if the error already has a stack trace
	var domainErr *Error
	if errors.As(err, &domainErr) && domainErr.Stack != "" {
		return err
	}

	// Capture stack trace
	const depth = 32
	var pcs [depth]uintptr
	n := runtime.Callers(2, pcs[:])
	frames := runtime.CallersFrames(pcs[:n])

	var stackBuilder strings.Builder
	for {
		frame, more := frames.Next()
		if !strings.Contains(frame.File, "runtime/") {
			fmt.Fprintf(&stackBuilder, "%s:%d %s\n", frame.File, frame.Line, frame.Function)
		}
		if !more {
			break
		}
	}

	// If it's already a domain error, just add the stack
	if errors.As(err, &domainErr) {
		domainErr.Stack = stackBuilder.String()
		return domainErr
	}

	// Otherwise, create a new domain error
	return &Error{
		Original: err,
		Stack:    stackBuilder.String(),
	}
}

// Wrap wraps an error with a message
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}

	// If it's already a domain error, update it
	var domainErr *Error
	if errors.As(err, &domainErr) {
		// Create a new error to avoid modifying the original
		return &Error{
			Original:  domainErr.Original,
			Domain:    domainErr.Domain,
			Code:      domainErr.Code,
			Message:   message,
			Operation: domainErr.Operation,
			Fields:    domainErr.Fields,
			Stack:     domainErr.Stack,
		}
	}

	// Otherwise, create a new domain error
	return &Error{
		Original: err,
		Message:  message,
	}
}

// WrapWithDomain wraps an error with a domain
func WrapWithDomain(err error, domain string) error {
	if err == nil {
		return nil
	}

	// If it's already a domain error, update it
	var domainErr *Error
	if errors.As(err, &domainErr) {
		// Create a new error to avoid modifying the original
		return &Error{
			Original:  domainErr.Original,
			Domain:    domain,
			Code:      domainErr.Code,
			Message:   domainErr.Message,
			Operation: domainErr.Operation,
			Fields:    domainErr.Fields,
			Stack:     domainErr.Stack,
		}
	}

	// Otherwise, create a new domain error
	return &Error{
		Original: err,
		Domain:   domain,
	}
}

// WrapWithOperation wraps an error with an operation
func WrapWithOperation(err error, operation string) error {
	if err == nil {
		return nil
	}

	// If it's already a domain error, update it
	var domainErr *Error
	if errors.As(err, &domainErr) {
		// Create a new error to avoid modifying the original
		return &Error{
			Original:  domainErr.Original,
			Domain:    domainErr.Domain,
			Code:      domainErr.Code,
			Message:   domainErr.Message,
			Operation: operation,
			Fields:    domainErr.Fields,
			Stack:     domainErr.Stack,
		}
	}

	// Otherwise, create a new domain error
	return &Error{
		Original:  err,
		Operation: operation,
	}
}

// WrapWithCode wraps an error with a code
func WrapWithCode(err error, code string) error {
	if err == nil {
		return nil
	}

	// If it's already a domain error, update it
	var domainErr *Error
	if errors.As(err, &domainErr) {
		// Create a new error to avoid modifying the original
		return &Error{
			Original:  domainErr.Original,
			Domain:    domainErr.Domain,
			Code:      code,
			Message:   domainErr.Message,
			Operation: domainErr.Operation,
			Fields:    domainErr.Fields,
			Stack:     domainErr.Stack,
		}
	}

	// Otherwise, create a new domain error
	return &Error{
		Original: err,
		Code:     code,
	}
}

// WrapWithField wraps an error with a field
func WrapWithField(err error, key string, value interface{}) error {
	if err == nil {
		return nil
	}

	// If it's already a domain error, update it
	var domainErr *Error
	if errors.As(err, &domainErr) {
		// Create a new error to avoid modifying the original
		newFields := make(map[string]interface{})
		for k, v := range domainErr.Fields {
			newFields[k] = v
		}
		if newFields == nil {
			newFields = make(map[string]interface{})
		}
		newFields[key] = value

		return &Error{
			Original:  domainErr.Original,
			Domain:    domainErr.Domain,
			Code:      domainErr.Code,
			Message:   domainErr.Message,
			Operation: domainErr.Operation,
			Fields:    newFields,
			Stack:     domainErr.Stack,
		}
	}

	// Otherwise, create a new domain error
	fields := make(map[string]interface{})
	fields[key] = value

	return &Error{
		Original: err,
		Fields:   fields,
	}
}

// E is a convenience function for creating domain errors
func E(args ...interface{}) error {
	if len(args) == 0 {
		return nil
	}

	e := &Error{}

	for _, arg := range args {
		switch a := arg.(type) {
		case string:
			// If we haven't set a message yet, set it
			if e.Message == "" {
				e.Message = a
			} else if e.Domain == "" {
				// If we have a message but no domain, set the domain
				e.Domain = a
			} else if e.Operation == "" {
				// If we have a message and domain but no operation, set the operation
				e.Operation = a
			} else if e.Code == "" {
				// If we have a message, domain, and operation but no code, set the code
				e.Code = a
			}
		case error:
			e.Original = a
		case map[string]interface{}:
			e.Fields = a
		}
	}

	return e
}
