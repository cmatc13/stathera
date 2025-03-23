// internal/orderbook/service.go
package orderbook

import (
	"context"
	"fmt"

	"github.com/cmatc13/stathera/pkg/service"
)

// OrderBookService wraps the RedisOrderBook as a Service
type OrderBookService struct {
	orderbook *RedisOrderBook
	status    service.Status
}

// NewOrderBookService creates a new orderbook service
func NewOrderBookService(redisAddr string) (*OrderBookService, error) {
	orderbook, err := NewRedisOrderBook(redisAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis orderbook: %w", err)
	}

	return &OrderBookService{
		orderbook: orderbook,
		status:    service.StatusStopped,
	}, nil
}

// Name returns the service name
func (s *OrderBookService) Name() string {
	return "orderbook"
}

// Start initializes and starts the service
func (s *OrderBookService) Start(ctx context.Context) error {
	s.status = service.StatusStarting

	// The orderbook doesn't have a long-running process to start
	// It's ready to use once initialized

	s.status = service.StatusRunning
	return nil
}

// Stop gracefully shuts down the service
func (s *OrderBookService) Stop(ctx context.Context) error {
	s.status = service.StatusStopping

	if s.orderbook != nil {
		if err := s.orderbook.Close(); err != nil {
			return fmt.Errorf("error closing orderbook: %w", err)
		}
	}

	s.status = service.StatusStopped
	return nil
}

// Status returns the current service status
func (s *OrderBookService) Status() service.Status {
	return s.status
}

// Health performs a health check
func (s *OrderBookService) Health() error {
	if s.status != service.StatusRunning {
		return fmt.Errorf("service not running")
	}

	// In a real implementation, you would check the health of the orderbook
	// For example, check if it can connect to Redis

	return nil
}

// Dependencies returns a list of services this service depends on
func (s *OrderBookService) Dependencies() []string {
	return []string{} // No dependencies
}

// GetOrderBook provides access to the underlying orderbook
func (s *OrderBookService) GetOrderBook() *RedisOrderBook {
	return s.orderbook
}
