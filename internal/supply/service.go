// internal/supply/service.go
package supply

import (
	"context"
	"fmt"

	"github.com/cmatc13/stathera/internal/processor"
	"github.com/cmatc13/stathera/pkg/service"
)

// SupplyManagerService wraps the SupplyManager as a Service
type SupplyManagerService struct {
	manager *SupplyManager
	status  service.Status
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewSupplyManagerService creates a new supply manager service
func NewSupplyManagerService(
	redisAddr string,
	minInflation float64,
	maxInflation float64,
	maxStepSize float64,
	reserveAddress string,
	txProcessor processor.TransactionProcessor,
) (*SupplyManagerService, error) {
	manager, err := NewSupplyManager(
		redisAddr,
		minInflation,
		maxInflation,
		maxStepSize,
		reserveAddress,
		txProcessor,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create supply manager: %w", err)
	}

	return &SupplyManagerService{
		manager: manager,
		status:  service.StatusStopped,
	}, nil
}

// Name returns the service name
func (s *SupplyManagerService) Name() string {
	return "supply-manager"
}

// Start initializes and starts the service
func (s *SupplyManagerService) Start(ctx context.Context) error {
	s.status = service.StatusStarting

	// Create a new context with cancellation for the service
	s.ctx, s.cancel = context.WithCancel(ctx)

	// Start scheduled tasks
	s.manager.StartScheduledTasks(s.ctx)

	s.status = service.StatusRunning
	return nil
}

// Stop gracefully shuts down the service
func (s *SupplyManagerService) Stop(ctx context.Context) error {
	s.status = service.StatusStopping

	// Cancel the context to stop scheduled tasks
	if s.cancel != nil {
		s.cancel()
	}

	// Close the manager
	if s.manager != nil {
		if err := s.manager.Close(); err != nil {
			return fmt.Errorf("error closing supply manager: %w", err)
		}
	}

	s.status = service.StatusStopped
	return nil
}

// Status returns the current service status
func (s *SupplyManagerService) Status() service.Status {
	return s.status
}

// Health performs a health check
func (s *SupplyManagerService) Health() error {
	if s.status != service.StatusRunning {
		return fmt.Errorf("service not running")
	}

	// In a real implementation, you would check the health of the supply manager
	// For example, check if it can connect to Redis

	return nil
}

// Dependencies returns a list of services this service depends on
func (s *SupplyManagerService) Dependencies() []string {
	return []string{"transaction-processor"}
}

// GetSupplyManager provides access to the underlying supply manager
func (s *SupplyManagerService) GetSupplyManager() *SupplyManager {
	return s.manager
}
