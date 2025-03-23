// internal/processor/service.go
package processor

import (
	"context"
	"fmt"

	"github.com/cmatc13/stathera/pkg/service"
)

// TransactionProcessorService wraps the TransactionProcessor as a Service
type TransactionProcessorService struct {
	processor *TransactionProcessor
	status    service.Status
}

// NewTransactionProcessorService creates a new transaction processor service
func NewTransactionProcessorService(processor *TransactionProcessor) *TransactionProcessorService {
	return &TransactionProcessorService{
		processor: processor,
		status:    service.StatusStopped,
	}
}

// Name returns the service name
func (s *TransactionProcessorService) Name() string {
	return "transaction-processor"
}

// Start initializes and starts the service
func (s *TransactionProcessorService) Start(ctx context.Context) error {
	s.status = service.StatusStarting

	// Start the transaction processor
	go s.processor.Start()

	s.status = service.StatusRunning
	return nil
}

// Stop gracefully shuts down the service
func (s *TransactionProcessorService) Stop(ctx context.Context) error {
	s.status = service.StatusStopping

	// The processor will be stopped via context cancellation
	// which is handled in the main function

	s.status = service.StatusStopped
	return nil
}

// Status returns the current service status
func (s *TransactionProcessorService) Status() service.Status {
	return s.status
}

// Health performs a health check
func (s *TransactionProcessorService) Health() error {
	if s.status != service.StatusRunning {
		return fmt.Errorf("service not running")
	}

	// In a real implementation, you would check the health of the processor
	// For example, check if it can connect to Redis and Kafka

	return nil
}

// Dependencies returns a list of services this service depends on
func (s *TransactionProcessorService) Dependencies() []string {
	return []string{} // No dependencies
}
