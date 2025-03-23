// pkg/service/service.go
package service

import (
	"context"
)

// Status represents the current state of a service
type Status string

const (
	StatusStopped  Status = "STOPPED"
	StatusStarting Status = "STARTING"
	StatusRunning  Status = "RUNNING"
	StatusStopping Status = "STOPPING"
	StatusError    Status = "ERROR"
)

// Service defines the interface that all services must implement
type Service interface {
	// Name returns the service name
	Name() string

	// Start initializes and starts the service
	Start(ctx context.Context) error

	// Stop gracefully shuts down the service
	Stop(ctx context.Context) error

	// Status returns the current service status
	Status() Status

	// Health performs a health check and returns error if unhealthy
	Health() error

	// Dependencies returns a list of services this service depends on
	Dependencies() []string
}
