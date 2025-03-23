// internal/api/service.go
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cmatc13/stathera/internal/orderbook"
	"github.com/cmatc13/stathera/internal/processor"
	"github.com/cmatc13/stathera/pkg/config"
	"github.com/cmatc13/stathera/pkg/health"
	"github.com/cmatc13/stathera/pkg/logging"
	"github.com/cmatc13/stathera/pkg/metrics"
	"github.com/cmatc13/stathera/pkg/service"
	txproc "github.com/cmatc13/stathera/pkg/transaction"
)

// APIService wraps the API server as a Service
type APIService struct {
	server           *Server
	config           *config.Config
	txProcessor      txproc.Processor
	orderbook        *orderbook.OrderBookService
	status           service.Status
	logger           *logging.Logger
	metricsCollector *metrics.Metrics
	healthRegistry   *health.Registry
	metricsServer    *http.Server
	healthServer     *http.Server
}

// NewAPIService creates a new API service
func NewAPIService(
	cfg *config.Config,
	txProcessor *processor.TransactionProcessor,
	orderbook *orderbook.OrderBookService,
) *APIService {
	// Set up structured logger
	logCfg := logging.Config{
		Level:       logging.LogLevel(cfg.Log.Level),
		Output:      logging.DefaultConfig().Output,
		ServiceName: "api-service",
		Environment: cfg.Log.Environment,
	}
	logger := logging.New(logCfg)

	// Set up metrics
	metricsCfg := metrics.Config{
		Namespace:   cfg.Metrics.Namespace,
		Subsystem:   "api",
		ServiceName: "api-service",
	}
	metricsCollector := metrics.New(metricsCfg)

	// Set up health registry
	healthRegistry := health.NewRegistry(logger)

	return &APIService{
		config:           cfg,
		txProcessor:      txProcessor,
		orderbook:        orderbook,
		status:           service.StatusStopped,
		logger:           logger,
		metricsCollector: metricsCollector,
		healthRegistry:   healthRegistry,
	}
}

// Name returns the service name
func (s *APIService) Name() string {
	return "api"
}

// Start initializes and starts the service
func (s *APIService) Start(ctx context.Context) error {
	s.status = service.StatusStarting
	s.logger.Info("Starting API service")

	// Initialize the API server
	s.server = NewServer(s.config, s.txProcessor, s.orderbook.GetOrderBook())

	// Start the server
	go s.server.Start()

	// Record service start in metrics
	s.metricsCollector.ServiceLastStarted.Set(float64(time.Now().Unix()))

	// Start recording uptime
	uptimeDone := make(chan struct{})
	s.metricsCollector.RecordUptime(uptimeDone)

	s.status = service.StatusRunning
	s.logger.Info("API service started successfully")
	return nil
}

// Stop gracefully shuts down the service
func (s *APIService) Stop(ctx context.Context) error {
	s.status = service.StatusStopping
	s.logger.Info("Stopping API service")

	if s.server != nil {
		s.server.Shutdown(ctx)
	}

	s.status = service.StatusStopped
	s.logger.Info("API service stopped successfully")
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

	// Check if the server is responding
	if s.server == nil {
		return fmt.Errorf("server not initialized")
	}

	// In a real implementation, you would make a request to the health endpoint
	// For now, we'll just check if the server is initialized
	return nil
}

// Dependencies returns a list of services this service depends on
func (s *APIService) Dependencies() []string {
	return []string{"transaction-processor", "orderbook"}
}

// GetMetrics returns the metrics collector for this service
func (s *APIService) GetMetrics() *metrics.Metrics {
	return s.metricsCollector
}

// GetHealthRegistry returns the health registry for this service
func (s *APIService) GetHealthRegistry() *health.Registry {
	return s.healthRegistry
}
