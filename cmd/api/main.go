// cmd/api/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/cmatc13/stathera/internal/api"
	"github.com/cmatc13/stathera/internal/orderbook"
	"github.com/cmatc13/stathera/internal/processor"
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
		fmt.Printf("Configuration loaded from file: %s\n", *configFile)
	} else if len(os.Getenv("STATHERA_ENV")) > 0 {
		fmt.Println("Configuration loaded from environment variables")
	} else {
		fmt.Println("Configuration loaded from defaults")
	}

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize transaction processor
	txProcessor, err := processor.NewTransactionProcessor(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to initialize transaction processor: %v", err)
	}

	// Start transaction processor
	go txProcessor.Start()

	// Initialize orderbook
	orderbookService, err := orderbook.NewRedisOrderBook(cfg.Redis.Address)
	if err != nil {
		log.Fatalf("Failed to initialize orderbook: %v", err)
	}

	// Initialize and start API server
	apiServer := api.NewServer(cfg, txProcessor, orderbookService)
	go apiServer.Start()

	// Handle graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs

	log.Println("Shutting down gracefully...")
	cancel()
	apiServer.Shutdown(ctx)
	log.Println("Shutdown complete")
}
