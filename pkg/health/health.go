// Package health provides health check capabilities for the application.
package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cmatc13/stathera/pkg/logging"
)

// Status represents the health status of a component.
type Status string

const (
	// StatusUp indicates the component is healthy.
	StatusUp Status = "UP"
	// StatusDown indicates the component is unhealthy.
	StatusDown Status = "DOWN"
	// StatusUnknown indicates the component's health is unknown.
	StatusUnknown Status = "UNKNOWN"
)

// Check represents a health check for a component.
type Check struct {
	// Name is the name of the component being checked.
	Name string
	// Status is the health status of the component.
	Status Status
	// Message is an optional message providing more details about the health status.
	Message string
	// LastChecked is the time when the component was last checked.
	LastChecked time.Time
	// Error is an optional error that occurred during the health check.
	Error error
}

// MarshalJSON implements the json.Marshaler interface.
func (c Check) MarshalJSON() ([]byte, error) {
	var errorStr string
	if c.Error != nil {
		errorStr = c.Error.Error()
	}

	return json.Marshal(struct {
		Name        string    `json:"name"`
		Status      Status    `json:"status"`
		Message     string    `json:"message,omitempty"`
		LastChecked time.Time `json:"last_checked"`
		Error       string    `json:"error,omitempty"`
	}{
		Name:        c.Name,
		Status:      c.Status,
		Message:     c.Message,
		LastChecked: c.LastChecked,
		Error:       errorStr,
	})
}

// Checker defines a function that performs a health check.
type Checker func(ctx context.Context) Check

// Registry manages health checks for the application.
type Registry struct {
	checks map[string]Checker
	mutex  sync.RWMutex
	logger *logging.Logger
}

// NewRegistry creates a new health check registry.
func NewRegistry(logger *logging.Logger) *Registry {
	return &Registry{
		checks: make(map[string]Checker),
		logger: logger,
	}
}

// Register adds a health check to the registry.
func (r *Registry) Register(name string, checker Checker) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.checks[name] = checker
	r.logger.Info("Registered health check", "name", name)
}

// Unregister removes a health check from the registry.
func (r *Registry) Unregister(name string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	delete(r.checks, name)
	r.logger.Info("Unregistered health check", "name", name)
}

// RunChecks runs all registered health checks.
func (r *Registry) RunChecks(ctx context.Context) map[string]Check {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	results := make(map[string]Check)
	for name, checker := range r.checks {
		r.logger.Debug("Running health check", "name", name)
		results[name] = checker(ctx)
	}

	return results
}

// IsHealthy returns true if all health checks are passing.
func (r *Registry) IsHealthy(ctx context.Context) bool {
	checks := r.RunChecks(ctx)
	for _, check := range checks {
		if check.Status != StatusUp {
			return false
		}
	}
	return true
}

// Handler returns an HTTP handler for health checks.
func (r *Registry) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		checks := r.RunChecks(ctx)

		// Determine overall status
		status := StatusUp
		for _, check := range checks {
			if check.Status == StatusDown {
				status = StatusDown
				break
			} else if check.Status == StatusUnknown && status != StatusDown {
				status = StatusUnknown
			}
		}

		// Set HTTP status code based on health status
		if status == StatusDown {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else if status == StatusUnknown {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		// Build response
		response := struct {
			Status    Status           `json:"status"`
			Timestamp time.Time        `json:"timestamp"`
			Checks    map[string]Check `json:"checks"`
		}{
			Status:    status,
			Timestamp: time.Now(),
			Checks:    checks,
		}

		// Write response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			r.logger.Error("Failed to encode health check response", "error", err)
		}
	})
}

// ServiceChecker creates a health check for a service.
func ServiceChecker(serviceName string, checkFn func(ctx context.Context) error) Checker {
	return func(ctx context.Context) Check {
		check := Check{
			Name:        serviceName,
			Status:      StatusUnknown,
			LastChecked: time.Now(),
		}

		err := checkFn(ctx)
		if err != nil {
			check.Status = StatusDown
			check.Error = err
			check.Message = fmt.Sprintf("Service %s is unhealthy: %v", serviceName, err)
		} else {
			check.Status = StatusUp
			check.Message = fmt.Sprintf("Service %s is healthy", serviceName)
		}

		return check
	}
}

// RedisChecker creates a health check for Redis.
func RedisChecker(redisAddr string, pingFn func(ctx context.Context) error) Checker {
	return func(ctx context.Context) Check {
		check := Check{
			Name:        "redis",
			Status:      StatusUnknown,
			LastChecked: time.Now(),
		}

		err := pingFn(ctx)
		if err != nil {
			check.Status = StatusDown
			check.Error = err
			check.Message = fmt.Sprintf("Redis at %s is unhealthy: %v", redisAddr, err)
		} else {
			check.Status = StatusUp
			check.Message = fmt.Sprintf("Redis at %s is healthy", redisAddr)
		}

		return check
	}
}

// KafkaChecker creates a health check for Kafka.
func KafkaChecker(kafkaBrokers string, checkFn func(ctx context.Context) error) Checker {
	return func(ctx context.Context) Check {
		check := Check{
			Name:        "kafka",
			Status:      StatusUnknown,
			LastChecked: time.Now(),
		}

		err := checkFn(ctx)
		if err != nil {
			check.Status = StatusDown
			check.Error = err
			check.Message = fmt.Sprintf("Kafka at %s is unhealthy: %v", kafkaBrokers, err)
		} else {
			check.Status = StatusUp
			check.Message = fmt.Sprintf("Kafka at %s is healthy", kafkaBrokers)
		}

		return check
	}
}

// DependencyChecker creates a health check for a dependency.
func DependencyChecker(dependencyName string, checkFn func(ctx context.Context) error) Checker {
	return func(ctx context.Context) Check {
		check := Check{
			Name:        dependencyName,
			Status:      StatusUnknown,
			LastChecked: time.Now(),
		}

		err := checkFn(ctx)
		if err != nil {
			check.Status = StatusDown
			check.Error = err
			check.Message = fmt.Sprintf("Dependency %s is unhealthy: %v", dependencyName, err)
		} else {
			check.Status = StatusUp
			check.Message = fmt.Sprintf("Dependency %s is healthy", dependencyName)
		}

		return check
	}
}
