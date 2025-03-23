// cmd/supply-manager/main.go
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cmatc13/stathera/internal/processor"
	"github.com/cmatc13/stathera/internal/supply"
	"github.com/cmatc13/stathera/pkg/config"
)

func main() {
	// Define command-line flags
	configFile := flag.String("config", "", "Path to configuration file")
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

	// Print configuration source for debugging
	if *configFile != "" {
		log.Printf("Configuration loaded from file: %s", *configFile)
	} else if len(os.Getenv("STATHERA_ENV")) > 0 {
		log.Println("Configuration loaded from environment variables")
	} else {
		log.Println("Configuration loaded from defaults")
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize transaction processor (needed for submitting supply increase txs)
	txProcessor, err := processor.NewTransactionProcessor(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize transaction processor: %v", err)
	}

	// Initialize supply manager
	supplyManager, err := supply.NewSupplyManager(
		cfg.Redis.Address,
		cfg.Supply.MinInflation,
		cfg.Supply.MaxInflation,
		cfg.Supply.MaxStepSize,
		cfg.Supply.ReserveAddress,
		txProcessor,
	)
	if err != nil {
		log.Fatalf("Failed to initialize supply manager: %v", err)
	}
	defer supplyManager.Close()

	// Start scheduled tasks
	supplyManager.StartScheduledTasks(ctx)

	log.Println("Supply manager started")

	// Handle graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	log.Println("Shutting down gracefully...")
	cancel()
	log.Println("Shutdown complete")
}
