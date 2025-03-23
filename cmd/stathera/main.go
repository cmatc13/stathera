// Package main provides the main entry point for the Stathera application.
// It initializes and coordinates all services using the service registry pattern.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cmatc13/stathera/internal/api"
	"github.com/cmatc13/stathera/internal/orderbook"
	"github.com/cmatc13/stathera/internal/processor"
	"github.com/cmatc13/stathera/internal/supply"
	"github.com/cmatc13/stathera/pkg/config"
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

	// Set up logger
	logger := log.New(os.Stdout, "[STATHERA] ", log.LstdFlags)

	// Print configuration source for debugging
	if *configFile != "" {
		logger.Printf("Configuration loaded from file: %s", *configFile)
	} else if len(os.Getenv("STATHERA_ENV")) > 0 {
		logger.Println("Configuration loaded from environment variables")
	} else {
		logger.Println("Configuration loaded from defaults")
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create service registry
	registry := service.NewRegistry(logger)

	// Initialize and register services
	logger.Println("Initializing services...")

	// Initialize and register transaction processor service
	txProcessor, err := processor.NewTransactionProcessor(ctx, cfg)
	if err != nil {
		logger.Fatalf("Failed to initialize transaction processor: %v", err)
	}
	txProcessorService := processor.NewTransactionProcessorService(txProcessor)
	if err := registry.Register(txProcessorService); err != nil {
		logger.Fatalf("Failed to register transaction processor service: %v", err)
	}

	// Initialize and register orderbook service
	orderbookService, err := orderbook.NewOrderBookService(cfg.Redis.Address)
	if err != nil {
		logger.Fatalf("Failed to initialize orderbook: %v", err)
	}
	if err := registry.Register(orderbookService); err != nil {
		logger.Fatalf("Failed to register orderbook service: %v", err)
	}

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
		logger.Fatalf("Failed to initialize supply manager: %v", err)
	}
	if err := registry.Register(supplyManagerService); err != nil {
		logger.Fatalf("Failed to register supply manager service: %v", err)
	}

	// Initialize and register API service
	apiService := api.NewAPIService(cfg, txProcessor, orderbookService)
	if err := registry.Register(apiService); err != nil {
		logger.Fatalf("Failed to register API service: %v", err)
	}

	// Start all services
	logger.Println("Starting all services...")
	if err := registry.StartAll(ctx); err != nil {
		logger.Fatalf("Failed to start services: %v", err)
	}
	logger.Println("All services started successfully")

	// Handle graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	logger.Println("Shutting down gracefully...")
	cancel()

	// Stop all services
	if err := registry.StopAll(context.Background()); err != nil {
		logger.Printf("Error during shutdown: %v", err)
	}

	logger.Println("Shutdown complete")
}
