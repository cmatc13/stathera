// Package main provides the main entry point for the Stathera application.
// It initializes and coordinates all services using the service registry pattern.
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cmatc13/stathera/internal/api"
	"github.com/cmatc13/stathera/internal/orderbook"
	"github.com/cmatc13/stathera/internal/processor"
	"github.com/cmatc13/stathera/internal/supply"
	"github.com/cmatc13/stathera/pkg/config"
	"github.com/cmatc13/stathera/pkg/health"
	"github.com/cmatc13/stathera/pkg/logging"
	"github.com/cmatc13/stathera/pkg/metrics"
	"github.com/cmatc13/stathera/pkg/service"
)

// main is the entry point for the Stathera application.
// It initializes configuration, sets up the service registry,
// registers all services, starts them in dependency order,
// and handles graceful shutdown.
func main() {
	// Define command-line flags
	configFile := flag.String("config", "", "Path to configuration file")
	logLevel := flag.String("log-level", "", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Set up custom load options
	opts := config.DefaultLoadOptions()
	if *configFile != "" {
		opts.ConfigFile = *configFile
	}

	// Initialize configuration
	cfg, err := config.LoadWithOptions(opts)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Override log level if specified via command line
	if *logLevel != "" {
		cfg.Log.Level = *logLevel
	}

	// Set up structured logger
	logCfg := logging.Config{
		Level:       logging.LogLevel(cfg.Log.Level),
		Output:      os.Stdout,
		ServiceName: cfg.Log.ServiceName,
		Environment: cfg.Log.Environment,
	}
	logger := logging.New(logCfg)

	// Print configuration source for debugging
	if *configFile != "" {
		logger.Info("Configuration loaded from file", "file", *configFile)
	} else if len(os.Getenv("STATHERA_ENV")) > 0 {
		logger.Info("Configuration loaded from environment variables")
	} else {
		logger.Info("Configuration loaded from defaults")
	}

	// Set up metrics
	metricsCfg := metrics.Config{
		Namespace:   cfg.Metrics.Namespace,
		Subsystem:   "",
		ServiceName: cfg.Metrics.ServiceName,
	}
	metricsCollector := metrics.New(metricsCfg)

	// Set up health check registry
	healthRegistry := health.NewRegistry(logger)

	// Start metrics server if enabled
	if cfg.Metrics.Enabled {
		go startMetricsServer(cfg, metricsCollector, logger)
	}

	// Start health check server if enabled
	if cfg.Health.Enabled {
		go startHealthServer(cfg, healthRegistry, logger)
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start recording uptime
	uptimeDone := make(chan struct{})
	metricsCollector.RecordUptime(uptimeDone)
	defer close(uptimeDone)

	// Create service registry with standard logger for now
	// We'll need to update the service registry to accept our structured logger
	stdLogger := log.New(os.Stdout, "[STATHERA] ", log.LstdFlags)
	registry := service.NewRegistry(stdLogger)

	// Initialize and register services
	logger.Info("Initializing services...")

	// Initialize and register transaction processor service
	txProcessor, err := processor.NewTransactionProcessor(ctx, cfg)
	if err != nil {
		logger.Error("Failed to initialize transaction processor", "error", err)
		os.Exit(1)
	}
	txProcessorService := processor.NewTransactionProcessorService(txProcessor)
	if err := registry.Register(txProcessorService); err != nil {
		logger.Error("Failed to register transaction processor service", "error", err)
		os.Exit(1)
	}

	// Register health check for transaction processor
	healthRegistry.Register("transaction-processor", health.ServiceChecker("transaction-processor", func(ctx context.Context) error {
		return txProcessorService.Health()
	}))

	// Initialize and register orderbook service
	orderbookService, err := orderbook.NewOrderBookService(cfg.Redis.Address)
	if err != nil {
		logger.Error("Failed to initialize orderbook", "error", err)
		os.Exit(1)
	}
	if err := registry.Register(orderbookService); err != nil {
		logger.Error("Failed to register orderbook service", "error", err)
		os.Exit(1)
	}

	// Register health check for orderbook
	healthRegistry.Register("orderbook", health.ServiceChecker("orderbook", func(ctx context.Context) error {
		return orderbookService.Health()
	}))

	// Initialize and register supply manager service
	// Note: txProcessor implements the pkg/transaction.Processor interface
	supplyManagerService, err := supply.NewSupplyManagerService(
		cfg.Redis.Address,
		cfg.Supply.MinInflation,
		cfg.Supply.MaxInflation,
		cfg.Supply.MaxStepSize,
		cfg.Supply.ReserveAddress,
		txProcessor, // Pass the transaction processor directly
	)
	if err != nil {
		logger.Error("Failed to initialize supply manager", "error", err)
		os.Exit(1)
	}
	if err := registry.Register(supplyManagerService); err != nil {
		logger.Error("Failed to register supply manager service", "error", err)
		os.Exit(1)
	}

	// Register health check for supply manager
	healthRegistry.Register("supply-manager", health.ServiceChecker("supply-manager", func(ctx context.Context) error {
		return supplyManagerService.Health()
	}))

	// Initialize and register API service
	apiService := api.NewAPIService(cfg, txProcessor, orderbookService)
	if err := registry.Register(apiService); err != nil {
		logger.Error("Failed to register API service", "error", err)
		os.Exit(1)
	}

	// Register health check for API
	healthRegistry.Register("api", health.ServiceChecker("api", func(ctx context.Context) error {
		return apiService.Health()
	}))

	// Register Redis health check
	healthRegistry.Register("redis", health.RedisChecker(cfg.Redis.Address, func(ctx context.Context) error {
		// This is a placeholder - in a real implementation, you would ping Redis
		// For now, we'll just check if the Redis address is valid
		return nil
	}))

	// Register Kafka health check
	healthRegistry.Register("kafka", health.KafkaChecker(cfg.Kafka.Brokers, func(ctx context.Context) error {
		// This is a placeholder - in a real implementation, you would check Kafka connectivity
		// For now, we'll just check if the Kafka brokers are valid
		return nil
	}))

	// Start all services
	logger.Info("Starting all services...")
	if err := registry.StartAll(ctx); err != nil {
		logger.Error("Failed to start services", "error", err)
		os.Exit(1)
	}
	logger.Info("All services started successfully")

	// Handle graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	logger.Info("Shutting down gracefully...")
	cancel()

	// Stop all services
	if err := registry.StopAll(context.Background()); err != nil {
		logger.Error("Error during shutdown", "error", err)
	}

	logger.Info("Shutdown complete")
}

// startMetricsServer starts a server to expose Prometheus metrics
func startMetricsServer(cfg *config.Config, metricsCollector *metrics.Metrics, logger *logging.Logger) {
	addr := fmt.Sprintf(":%s", cfg.Metrics.Port)
	mux := http.NewServeMux()
	mux.Handle(cfg.Metrics.Endpoint, metricsCollector.Handler())

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Record the start time for metrics
	metricsCollector.ServiceLastStarted.Set(float64(time.Now().Unix()))

	logger.Info("Starting metrics server", "addr", addr, "endpoint", cfg.Metrics.Endpoint)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Metrics server failed", "error", err)
	}
}

// startHealthServer starts a server to expose health check endpoints
func startHealthServer(cfg *config.Config, healthRegistry *health.Registry, logger *logging.Logger) {
	addr := fmt.Sprintf(":%s", cfg.Health.Port)
	mux := http.NewServeMux()
	mux.Handle(cfg.Health.Endpoint, healthRegistry.Handler())

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	logger.Info("Starting health check server", "addr", addr, "endpoint", cfg.Health.Endpoint)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("Health check server failed", "error", err)
	}
}
