// Package service provides interfaces and utilities for managing service lifecycle.
package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// Registry manages all services and their lifecycle.
// It handles service registration, dependency resolution, and coordinated
// startup and shutdown of services.
type Registry struct {
	services map[string]Service
	mutex    sync.RWMutex
	logger   *log.Logger
}

// NewRegistry creates a new service registry with the provided logger.
// The registry is used to manage the lifecycle of all services in the application.
func NewRegistry(logger *log.Logger) *Registry {
	return &Registry{
		services: make(map[string]Service),
		logger:   logger,
	}
}

// Register adds a service to the registry.
// It returns an error if a service with the same name is already registered.
func (r *Registry) Register(service Service) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	name := service.Name()
	if _, exists := r.services[name]; exists {
		return fmt.Errorf("service %s is already registered", name)
	}

	r.services[name] = service
	r.logger.Printf("Service registered: %s", name)
	return nil
}

// Get returns a service by name.
// It returns an error if the service is not found.
func (r *Registry) Get(name string) (Service, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	service, exists := r.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}

	return service, nil
}

// StartAll starts all services in dependency order.
// It builds a dependency graph, performs a topological sort to determine
// the correct startup order, and starts each service in that order.
// It waits for each service to become healthy before starting the next one.
func (r *Registry) StartAll(ctx context.Context) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Build dependency graph and detect cycles
	graph := buildDependencyGraph(r.services)
	order, err := topologicalSort(graph)
	if err != nil {
		return fmt.Errorf("dependency cycle detected: %w", err)
	}

	// Start services in order
	for _, name := range order {
		service := r.services[name]
		r.logger.Printf("Starting service: %s", name)

		if err := service.Start(ctx); err != nil {
			r.logger.Printf("Failed to start service %s: %v", name, err)
			return fmt.Errorf("failed to start service %s: %w", name, err)
		}

		// Wait for service to be healthy
		if err := r.waitForHealth(ctx, name); err != nil {
			return err
		}
	}

	return nil
}

// StopAll stops all services in reverse dependency order.
// It builds a dependency graph, performs a topological sort, reverses the order,
// and stops each service in that order. This ensures that services are stopped
// in the correct order to avoid dependency issues.
func (r *Registry) StopAll(ctx context.Context) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Build dependency graph and detect cycles
	graph := buildDependencyGraph(r.services)
	order, err := topologicalSort(graph)
	if err != nil {
		return fmt.Errorf("dependency cycle detected: %w", err)
	}

	// Reverse the order for stopping
	for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
		order[i], order[j] = order[j], order[i]
	}

	// Stop services in reverse order
	for _, name := range order {
		service := r.services[name]
		r.logger.Printf("Stopping service: %s", name)

		if err := service.Stop(ctx); err != nil {
			r.logger.Printf("Error stopping service %s: %v", name, err)
			// Continue stopping other services
		}
	}

	return nil
}

// HealthCheck performs health checks on all services.
// It returns a map of service names to health check results (nil if healthy, error if not).
func (r *Registry) HealthCheck() map[string]error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	results := make(map[string]error)
	for name, service := range r.services {
		results[name] = service.Health()
	}

	return results
}

// waitForHealth waits for a service to become healthy.
// It polls the service's Health method until it returns nil or a timeout occurs.
func (r *Registry) waitForHealth(ctx context.Context, name string) error {
	service, err := r.Get(name)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(30 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for service %s to become healthy", name)
		case <-ticker.C:
			if err := service.Health(); err == nil {
				return nil
			}
		}
	}
}

// buildDependencyGraph creates a graph representation of service dependencies.
// The graph is a map where keys are service names and values are lists of
// services that the key service depends on.
func buildDependencyGraph(services map[string]Service) map[string][]string {
	graph := make(map[string][]string)

	for name, service := range services {
		graph[name] = service.Dependencies()
	}

	return graph
}

// topologicalSort performs a topological sort on the dependency graph
// and returns the sorted service names. It detects cycles in the dependency
// graph and returns an error if a cycle is found.
func topologicalSort(graph map[string][]string) ([]string, error) {
	// Create a map to track visited nodes
	visited := make(map[string]bool)
	// Create a map to track nodes in the current recursion stack
	temp := make(map[string]bool)
	// Create a list to store the sorted nodes
	order := make([]string, 0, len(graph))

	// Define a recursive visit function
	var visit func(node string) error
	visit = func(node string) error {
		// If node is in temp, we have a cycle
		if temp[node] {
			return fmt.Errorf("dependency cycle detected involving service %s", node)
		}

		// If node is already visited, skip it
		if visited[node] {
			return nil
		}

		// Mark node as temporarily visited
		temp[node] = true

		// Visit all dependencies
		for _, dep := range graph[node] {
			// Skip if dependency doesn't exist (might be external)
			if _, exists := graph[dep]; !exists {
				continue
			}

			if err := visit(dep); err != nil {
				return err
			}
		}

		// Mark node as visited
		visited[node] = true
		// Remove from temp
		temp[node] = false

		// Add to order
		order = append(order, node)

		return nil
	}

	// Visit all nodes
	for node := range graph {
		if !visited[node] {
			if err := visit(node); err != nil {
				return nil, err
			}
		}
	}

	// Reverse the order (topological sort gives reverse dependency order)
	for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
		order[i], order[j] = order[j], order[i]
	}

	return order, nil
}
