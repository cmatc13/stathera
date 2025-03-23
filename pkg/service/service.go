// Package service provides interfaces and utilities for managing service lifecycle.
// It defines a common Service interface that all services must implement, along with
// a registry for coordinating service startup and shutdown.
package service

import (
	"context"
)

// Status represents the current state of a service.
type Status string

const (
	// StatusStopped indicates the service is not running.
	StatusStopped Status = "STOPPED"
	// StatusStarting indicates the service is in the process of starting.
	StatusStarting Status = "STARTING"
	// StatusRunning indicates the service is running normally.
	StatusRunning Status = "RUNNING"
	// StatusStopping indicates the service is in the process of stopping.
	StatusStopping Status = "STOPPING"
	// StatusError indicates the service encountered an error.
	StatusError Status = "ERROR"
)

// Service defines the interface that all services must implement.
// This interface provides methods for managing the lifecycle of a service,
// including starting, stopping, health checking, and dependency management.
type Service interface {
	// Name returns the service name.
	Name() string

	// Start initializes and starts the service.
	// It should be non-blocking and return quickly, with any long-running
	// operations started in separate goroutines.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the service.
	// It should ensure all resources are properly released and any
	// ongoing operations are completed or terminated gracefully.
	Stop(ctx context.Context) error

	// Status returns the current service status.
	// This can be used to check if the service is running, starting, stopping, etc.
	Status() Status

	// Health performs a health check and returns error if unhealthy.
	// This is used by the registry to determine if the service is functioning properly.
	Health() error

	// Dependencies returns a list of services this service depends on.
	// The registry uses this information to determine the order in which
	// services should be started and stopped.
	Dependencies() []string
}
