// internal/api/service.go
package api

import (
	"context"
	"fmt"

	"github.com/cmatc13/stathera/internal/orderbook"
	"github.com/cmatc13/stathera/internal/processor"
	"github.com/cmatc13/stathera/pkg/config"
	"github.com/cmatc13/stathera/pkg/service"
	txproc "github.com/cmatc13/stathera/pkg/transaction"
)

// APIService wraps the API server as a Service
type APIService struct {
	server      *Server
	config      *config.Config
	txProcessor txproc.Processor
	orderbook   *orderbook.OrderBookService
	status      service.Status
}

// NewAPIService creates a new API service
func NewAPIService(
	cfg *config.Config,
	txProcessor *processor.TransactionProcessor,
	orderbook *orderbook.OrderBookService,
) *APIService {
	return &APIService{
		config:      cfg,
		txProcessor: txProcessor,
		orderbook:   orderbook,
		status:      service.StatusStopped,
	}
}

// Name returns the service name
func (s *APIService) Name() string {
	return "api"
}

// Start initializes and starts the service
func (s *APIService) Start(ctx context.Context) error {
	s.status = service.StatusStarting

	// Initialize the API server
	s.server = NewServer(s.config, s.txProcessor, s.orderbook.GetOrderBook())

	// Start the server
	go s.server.Start()

	s.status = service.StatusRunning
	return nil
}

// Stop gracefully shuts down the service
func (s *APIService) Stop(ctx context.Context) error {
	s.status = service.StatusStopping

	if s.server != nil {
		s.server.Shutdown(ctx)
	}

	s.status = service.StatusStopped
	return nil
}

// Status returns the current service status
func (s *APIService) Status() service.Status {
	return s.status
}

// Health performs a health check
func (s *APIService) Health() error {
	if s.status != service.StatusRunning {
		return fmt.Errorf("service not running")
	}

	// In a real implementation, you would check the health of the API server
	// For example, make a request to the health endpoint

	return nil
}

// Dependencies returns a list of services this service depends on
func (s *APIService) Dependencies() []string {
	return []string{"transaction-processor", "orderbook"}
}
