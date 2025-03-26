// Package main provides the entry point for the Stathera API server.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cmatc13/stathera/api"
	"github.com/cmatc13/stathera/ledger"
	"github.com/cmatc13/stathera/settlement"
	"github.com/cmatc13/stathera/timeoracle"
	"github.com/cmatc13/stathera/transaction"
)

const (
	// Default values
	defaultInitialSupply  = 20000000000000.0 // ~$20 trillion USD M2 equivalent
	defaultMinInflation   = 1.5              // 1.5% annual inflation minimum
	defaultMaxInflation   = 3.0              // 3.0% annual inflation maximum
	defaultBatchSize      = 1000             // Number of transactions per settlement batch
	defaultSettleInterval = 5 * time.Minute  // Settlement interval
	defaultAPIPort        = 8080             // Default API port
)

func main() {
	// Parse command-line flags
	initialSupply := flag.Float64("initial-supply", defaultInitialSupply, "Initial monetary supply")
	minInflation := flag.Float64("min-inflation", defaultMinInflation, "Minimum annual inflation rate (%)")
	maxInflation := flag.Float64("max-inflation", defaultMaxInflation, "Maximum annual inflation rate (%)")
	batchSize := flag.Int("batch-size", defaultBatchSize, "Number of transactions per settlement batch")
	settleInterval := flag.Duration("settle-interval", defaultSettleInterval, "Settlement interval")
	reserveAddress := flag.String("reserve-address", "RESERVE", "Reserve account address")
	feeAddress := flag.String("fee-address", "FEES", "Fee collection address")
	apiPort := flag.Int("api-port", defaultAPIPort, "API server port")
	flag.Parse()

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize time oracle
	timeOracle, err := initializeTimeOracle()
	if err != nil {
		log.Fatalf("Failed to initialize time oracle: %v", err)
	}

	// Initialize ledger (Layer 1)
	canonicalLedger, err := ledger.NewLedger(*initialSupply, *minInflation, *maxInflation, timeOracle)
	if err != nil {
		log.Fatalf("Failed to initialize ledger: %v", err)
	}
	log.Printf("Ledger initialized with supply: %.2f", *initialSupply)

	// Initialize transaction engine (Layer 2)
	txEngine := transaction.NewTransactionEngine(timeOracle, *feeAddress)
	log.Printf("Transaction engine initialized")

	// Create system accounts
	createSystemAccounts(txEngine, *reserveAddress, *feeAddress)

	// Initialize settlement engine (Layer 3)
	settlementEngine := settlement.NewSettlementEngine(
		txEngine,
		canonicalLedger,
		timeOracle,
		*batchSize,
		*settleInterval,
	)
	log.Printf("Settlement engine initialized")

	// Start settlement process
	go settlementEngine.StartSettlementProcess(ctx)
	log.Printf("Settlement process started with interval: %v", *settleInterval)

	// Initialize API server
	apiServer := api.NewServer(
		txEngine,
		canonicalLedger,
		settlementEngine,
		timeOracle,
		*apiPort,
	)
	log.Printf("API server initialized on port %d", *apiPort)

	// Start API server in a goroutine
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Fatalf("API server failed: %v", err)
		}
	}()

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")

	// Stop API server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := apiServer.Stop(shutdownCtx); err != nil {
		log.Printf("Error stopping API server: %v", err)
	}

	// Cancel context to stop settlement process
	cancel()

	log.Println("Shutdown complete")
}

// initializeTimeOracle creates and initializes the time oracle
func initializeTimeOracle() (timeoracle.TimeOracle, error) {
	// Generate a secure random secret
	secret := make([]byte, 32)

	// In a real implementation, you would use crypto/rand
	// For simplicity, we're using a fixed secret here
	for i := range secret {
		secret[i] = byte(i)
	}

	// Create time oracle with 5 second max drift and 24 hour proof validity
	oracle, err := timeoracle.NewStandardTimeOracle(
		secret,
		5*time.Second,
		24*time.Hour,
	)
	if err != nil {
		return nil, err
	}

	return oracle, nil
}

// createSystemAccounts creates the necessary system accounts
func createSystemAccounts(txEngine *transaction.TransactionEngine, reserveAddress, feeAddress string) {
	// Generate dummy public keys for system accounts
	reservePubKey := make([]byte, 32)
	feePubKey := make([]byte, 32)

	// Create reserve account
	if err := txEngine.CreateAccount(reserveAddress, reservePubKey); err != nil {
		log.Printf("Reserve account already exists: %v", err)
	} else {
		log.Printf("Created reserve account: %s", reserveAddress)
	}

	// Create fee account
	if err := txEngine.CreateAccount(feeAddress, feePubKey); err != nil {
		log.Printf("Fee account already exists: %v", err)
	} else {
		log.Printf("Created fee account: %s", feeAddress)
	}
}
