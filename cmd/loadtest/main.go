// cmd/loadtest/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-redis/redis/v8"

	"github.com/cmatc13/stathera/internal/transaction"
	"github.com/cmatc13/stathera/internal/wallet"
)

// Command line flags
var (
	duration        = flag.Duration("duration", 1*time.Minute, "Test duration")
	numWallets      = flag.Int("wallets", 1000, "Number of wallets to use")
	concurrency     = flag.Int("concurrency", 100, "Number of concurrent clients")
	transactionRate = flag.Float64("rate", 1000, "Target transactions per second")
	redisAddr       = flag.String("redis", "", "Redis address (overrides config)")
	initialBalance  = flag.Float64("balance", 10000.0, "Initial balance for each wallet")
	configFile      = flag.String("config", "", "Path to configuration file")
)

// Statistics
type Stats struct {
	successCount uint64
	failureCount uint64
	latencySum   uint64
	latencyCount uint64
}

func main() {
	flag.Parse()

	// Print test configuration
	fmt.Printf("Load Test Configuration:\n")
	fmt.Printf("  Duration: %s\n", *duration)
	fmt.Printf("  Wallets: %d\n", *numWallets)
	fmt.Printf("  Concurrency: %d\n", *concurrency)
	fmt.Printf("  Target TPS: %.0f\n", *transactionRate)
	fmt.Printf("  Redis: %s\n", *redisAddr)
	fmt.Printf("  Initial Balance: %.2f\n", *initialBalance)

	// Create context that can be cancelled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-signals
		fmt.Println("\nShutting down...")
		cancel()
	}()

	// Create Redis client for setup
	redisClient := redis.NewClient(&redis.Options{
		Addr: *redisAddr,
	})
	defer redisClient.Close()

	// Generate wallets
	fmt.Printf("Generating %d test wallets...\n", *numWallets)
	wallets, err := generateWallets(*numWallets)
	if err != nil {
		log.Fatalf("Failed to generate wallets: %v", err)
	}

	// Set initial balances
	fmt.Printf("Setting initial balances to %.2f...\n", *initialBalance)
	err = setInitialBalances(ctx, redisClient, wallets, *initialBalance)
	if err != nil {
		log.Fatalf("Failed to set initial balances: %v", err)
	}

	// Create stats collector
	stats := &Stats{}

	// Start load test
	fmt.Printf("Starting load test for %s...\n", *duration)

	// Create timer for test duration
	testTimer := time.NewTimer(*duration)

	// Create context with timeout for the test duration
	testCtx, testCancel := context.WithTimeout(ctx, *duration)
	defer testCancel()

	// Create wait group for worker goroutines
	var wg sync.WaitGroup

	// Channel for controlling rate
	rateLimiter := make(chan struct{}, *concurrency*2)

	// Start rate limiter
	go func() {
		interval := time.Duration(float64(time.Second) / *transactionRate)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-testCtx.Done():
				return
			case <-ticker.C:
				select {
				case rateLimiter <- struct{}{}:
				default:
					// Channel is full, skip
				}
			}
		}
	}()

	// Start worker goroutines
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go worker(testCtx, i, redisClient, wallets, rateLimiter, stats, &wg)
	}

	// Start stats reporter
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastSuccessCount := uint64(0)
	lastFailureCount := uint64(0)
	startTime := time.Now()

	go func() {
		for {
			select {
			case <-testCtx.Done():
				return
			case <-ticker.C:
				successCount := atomic.LoadUint64(&stats.successCount)
				failureCount := atomic.LoadUint64(&stats.failureCount)
				latencySum := atomic.LoadUint64(&stats.latencySum)
				latencyCount := atomic.LoadUint64(&stats.latencyCount)

				successDelta := successCount - lastSuccessCount
				failureDelta := failureCount - lastFailureCount

				totalDelta := successDelta + failureDelta

				var avgLatency uint64
				if latencyCount > 0 {
					avgLatency = latencySum / latencyCount
				}

				elapsedSeconds := time.Since(startTime).Seconds()
				overallTPS := float64(successCount) / elapsedSeconds

				fmt.Printf("\rTPS: %.2f (Current: %d), Success: %d, Failure: %d, Avg Latency: %d µs",
					overallTPS, totalDelta, successCount, failureCount, avgLatency)

				lastSuccessCount = successCount
				lastFailureCount = failureCount
			}
		}
	}()

	// Wait for test to complete
	select {
	case <-testTimer.C:
		fmt.Println("\nTest duration reached")
	case <-ctx.Done():
		fmt.Println("\nTest interrupted")
	}

	testCancel()

	// Wait for all workers to finish
	wg.Wait()

	// Print final stats
	successCount := atomic.LoadUint64(&stats.successCount)
	failureCount := atomic.LoadUint64(&stats.failureCount)
	latencySum := atomic.LoadUint64(&stats.latencySum)
	latencyCount := atomic.LoadUint64(&stats.latencyCount)

	totalCount := successCount + failureCount
	successRate := float64(successCount) / float64(totalCount) * 100

	var avgLatency uint64
	if latencyCount > 0 {
		avgLatency = latencySum / latencyCount
	}

	elapsedSeconds := time.Since(startTime).Seconds()

	fmt.Printf("\n\nLoad Test Results:\n")
	fmt.Printf("  Test Duration: %.2f seconds\n", elapsedSeconds)
	fmt.Printf("  Total Transactions: %d\n", totalCount)
	fmt.Printf("  Successful Transactions: %d (%.2f%%)\n", successCount, successRate)
	fmt.Printf("  Failed Transactions: %d (%.2f%%)\n", failureCount, 100-successRate)
	fmt.Printf("  Average TPS: %.2f\n", float64(totalCount)/elapsedSeconds)
	fmt.Printf("  Average Latency: %d µs\n", avgLatency)
}

// worker processes transactions at the specified rate
func worker(ctx context.Context, id int, redisClient *redis.Client, wallets []*wallet.Wallet, rateLimiter <-chan struct{}, stats *Stats, wg *sync.WaitGroup) {
	defer wg.Done()

	// Create random number generator for this worker
	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))

	for {
		select {
		case <-ctx.Done():
			return
		case <-rateLimiter:
			// Process a transaction
			startTime := time.Now()

			// Select random sender and receiver
			senderIndex := r.Intn(len(wallets))
			receiverIndex := (senderIndex + 1 + r.Intn(len(wallets)-1)) % len(wallets)

			sender := wallets[senderIndex]
			receiver := wallets[receiverIndex]

			// Create transaction
			amount := 1.0 + 9.0*r.Float64() // Random amount between 1 and 10
			nonce := fmt.Sprintf("%d-%d-%d", time.Now().UnixNano(), id, r.Int())

			tx, err := transaction.NewTransaction(
				sender.Address,
				receiver.Address,
				amount,
				0.1, // Fixed fee
				transaction.Payment,
				nonce,
				fmt.Sprintf("Load test transaction from worker %d", id),
			)

			if err != nil {
				atomic.AddUint64(&stats.failureCount, 1)
				continue
			}

			// Sign transaction
			signData, err := tx.SignableData()
			if err != nil {
				atomic.AddUint64(&stats.failureCount, 1)
				continue
			}

			tx.Signature, err = sender.SignMessage(signData)
			if err != nil {
				atomic.AddUint64(&stats.failureCount, 1)
				continue
			}

			// Process transaction using Redis Lua script
			txJSON, err := tx.ToJSON()
			if err != nil {
				atomic.AddUint64(&stats.failureCount, 1)
				continue
			}

			// Define Lua script for atomic transaction processing
			luaScript := redis.NewScript(`
				-- Get current balances
				local senderBalance = tonumber(redis.call("GET", "balance:" .. ARGV[1]) or "0")
				local receiverBalance = tonumber(redis.call("GET", "balance:" .. ARGV[2]) or "0")
				local feeCollectorBalance = tonumber(redis.call("GET", "balance:fee_collector") or "0")
				
				-- Check if sender has sufficient balance
				local totalDebit = tonumber(ARGV[3]) + tonumber(ARGV[4])
				if senderBalance < totalDebit then
					return 0 -- Insufficient funds
				end
				
				-- Update balances
				redis.call("SET", "balance:" .. ARGV[1], senderBalance - totalDebit)
				redis.call("SET", "balance:" .. ARGV[2], receiverBalance + tonumber(ARGV[3]))
				redis.call("SET", "balance:fee_collector", feeCollectorBalance + tonumber(ARGV[4]))
				
				-- Store transaction data (in a real system)
				-- redis.call("SET", "tx:" .. ARGV[5], ARGV[6])
				
				return 1
			`)

			// Execute Lua script
			res, err := luaScript.Run(ctx, redisClient, []string{},
				sender.Address, receiver.Address, amount, 0.1, tx.ID, string(txJSON)).Result()

			elapsedMicros := time.Since(startTime).Microseconds()

			// Update stats
			if err != nil || res == int64(0) {
				atomic.AddUint64(&stats.failureCount, 1)
			} else {
				atomic.AddUint64(&stats.successCount, 1)
				atomic.AddUint64(&stats.latencySum, uint64(elapsedMicros))
				atomic.AddUint64(&stats.latencyCount, 1)
			}
		}
	}
}

// generateWallets generates a specified number of test wallets
func generateWallets(count int) ([]*wallet.Wallet, error) {
	wallets := make([]*wallet.Wallet, count)

	for i := 0; i < count; i++ {
		newWallet, err := wallet.NewWallet()
		if err != nil {
			return nil, fmt.Errorf("failed to generate wallet: %w", err)
		}
		wallets[i] = newWallet
	}

	return wallets, nil
}

// setInitialBalances sets the initial balance for all wallets
func setInitialBalances(ctx context.Context, client *redis.Client, wallets []*wallet.Wallet, balance float64) error {
	// Use pipeline for batch operations
	pipe := client.Pipeline()

	for _, w := range wallets {
		pipe.Set(ctx, "balance:"+w.Address, balance, 0)
	}

	// Create fee collector address
	pipe.Set(ctx, "balance:fee_collector", 0, 0)

	_, err := pipe.Exec(ctx)
	return err
}
