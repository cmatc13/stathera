// internal/supply/manager.go
package supply

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/cmatc13/stathera/internal/transaction"
	txproc "github.com/cmatc13/stathera/pkg/transaction"

	"github.com/go-redis/redis/v8"
)

const (
	// Key for storing total currency supply
	totalSupplyKey = "system:total_supply"

	// Key for storing current inflation rate
	inflationRateKey = "system:inflation_rate"

	// Key for storing last inflation adjustment timestamp
	lastInflationAdjustKey = "system:last_inflation_adjust"

	// Key for storing last supply increase timestamp
	lastSupplyIncreaseKey = "system:last_supply_increase"

	// Default initial supply (USD M2 equivalent, in arbitrary units)
	defaultInitialSupply = 20000000000000.0 // ~$20 trillion USD M2 equivalent

	// Day in seconds
	dayInSeconds = 86400

	// Year in seconds (approximate)
	yearInSeconds = 31536000
)

// SupplyManager handles the currency supply and inflation rate
type SupplyManager struct {
	client         *redis.Client
	ctx            context.Context
	minInflation   float64
	maxInflation   float64
	maxStepSize    float64
	reserveAddress string
	txProcessor    txproc.Processor
}

// NewSupplyManager creates a new supply manager
func NewSupplyManager(
	redisAddr string,
	minInflation float64,
	maxInflation float64,
	maxStepSize float64,
	reserveAddress string,
	txProcessor txproc.Processor,
) (*SupplyManager, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})

	ctx := context.Background()

	// Test connection
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	sm := &SupplyManager{
		client:         client,
		ctx:            ctx,
		minInflation:   minInflation,
		maxInflation:   maxInflation,
		maxStepSize:    maxStepSize,
		reserveAddress: reserveAddress,
		txProcessor:    txProcessor,
	}

	// Initialize supply and inflation rate if not already set
	if err := sm.initializeSupply(); err != nil {
		return nil, err
	}

	return sm, nil
}

// Close closes the Redis connection
func (sm *SupplyManager) Close() error {
	return sm.client.Close()
}

// initializeSupply sets up the initial supply and inflation rate if not already set
func (sm *SupplyManager) initializeSupply() error {
	// Check if total supply exists
	exists, err := sm.client.Exists(sm.ctx, totalSupplyKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check if total supply exists: %w", err)
	}

	if exists == 0 {
		// Set initial total supply
		err = sm.client.Set(sm.ctx, totalSupplyKey, defaultInitialSupply, 0).Err()
		if err != nil {
			return fmt.Errorf("failed to set initial total supply: %w", err)
		}

		log.Printf("Initialized total supply to %.2f", defaultInitialSupply)
	}

	// Check if inflation rate exists
	exists, err = sm.client.Exists(sm.ctx, inflationRateKey).Result()
	if err != nil {
		return fmt.Errorf("failed to check if inflation rate exists: %w", err)
	}

	if exists == 0 {
		// Set initial inflation rate to midpoint of range
		initialRate := (sm.minInflation + sm.maxInflation) / 2
		err = sm.client.Set(sm.ctx, inflationRateKey, initialRate, 0).Err()
		if err != nil {
			return fmt.Errorf("failed to set initial inflation rate: %w", err)
		}

		log.Printf("Initialized inflation rate to %.2f%%", initialRate)
	}

	// Initialize timestamps if they don't exist
	now := time.Now().Unix()

	_, err = sm.client.Get(sm.ctx, lastInflationAdjustKey).Result()
	if err == redis.Nil {
		sm.client.Set(sm.ctx, lastInflationAdjustKey, now, 0)
	}

	_, err = sm.client.Get(sm.ctx, lastSupplyIncreaseKey).Result()
	if err == redis.Nil {
		sm.client.Set(sm.ctx, lastSupplyIncreaseKey, now, 0)
	}

	return nil
}

// GetTotalSupply returns the current total supply
func (sm *SupplyManager) GetTotalSupply() (float64, error) {
	supply, err := sm.client.Get(sm.ctx, totalSupplyKey).Float64()
	if err != nil {
		return 0, fmt.Errorf("failed to get total supply: %w", err)
	}
	return supply, nil
}

// GetInflationRate returns the current annual inflation rate
func (sm *SupplyManager) GetInflationRate() (float64, error) {
	rate, err := sm.client.Get(sm.ctx, inflationRateKey).Float64()
	if err != nil {
		return 0, fmt.Errorf("failed to get inflation rate: %w", err)
	}
	return rate, nil
}

// AdjustInflationRate updates the inflation rate based on a random walk
// Returns the new inflation rate
func (sm *SupplyManager) AdjustInflationRate() (float64, error) {
	// Get current timestamp and last adjustment timestamp
	now := time.Now().Unix()
	lastAdjust, err := sm.client.Get(sm.ctx, lastInflationAdjustKey).Int64()
	if err != nil && err != redis.Nil {
		return 0, fmt.Errorf("failed to get last inflation adjustment timestamp: %w", err)
	}

	// Only adjust once per day
	if now-lastAdjust < dayInSeconds {
		// Return current rate without adjustment
		return sm.GetInflationRate()
	}

	// Get current inflation rate
	currentRate, err := sm.GetInflationRate()
	if err != nil {
		return 0, err
	}

	// Generate random step between -maxStepSize and +maxStepSize
	rand.Seed(time.Now().UnixNano())
	step := (2*rand.Float64() - 1) * sm.maxStepSize

	// Calculate new rate
	newRate := currentRate + step

	// Ensure rate stays within bounds
	if newRate < sm.minInflation {
		newRate = sm.minInflation
	}
	if newRate > sm.maxInflation {
		newRate = sm.maxInflation
	}

	// Store new rate
	err = sm.client.Set(sm.ctx, inflationRateKey, newRate, 0).Err()
	if err != nil {
		return 0, fmt.Errorf("failed to set new inflation rate: %w", err)
	}

	// Update last adjustment timestamp
	err = sm.client.Set(sm.ctx, lastInflationAdjustKey, now, 0).Err()
	if err != nil {
		return 0, fmt.Errorf("failed to update last inflation adjustment timestamp: %w", err)
	}

	log.Printf("Adjusted inflation rate from %.2f%% to %.2f%%", currentRate, newRate)

	return newRate, nil
}

// IncreaseSupply increases the total supply based on the current inflation rate
// This should be called periodically (e.g., daily or weekly)
func (sm *SupplyManager) IncreaseSupply() error {
	// Get current timestamp and last increase timestamp
	now := time.Now().Unix()
	lastIncrease, err := sm.client.Get(sm.ctx, lastSupplyIncreaseKey).Int64()
	if err != nil && err != redis.Nil {
		return fmt.Errorf("failed to get last supply increase timestamp: %w", err)
	}

	// Only increase supply once per day
	if now-lastIncrease < dayInSeconds {
		return nil
	}

	// Get current total supply
	currentSupply, err := sm.GetTotalSupply()
	if err != nil {
		return err
	}

	// Get current inflation rate
	inflationRate, err := sm.GetInflationRate()
	if err != nil {
		return err
	}

	// Calculate daily inflation rate (annual rate / 365)
	dailyRate := inflationRate / 365.0

	// Calculate supply increase
	increaseAmount := currentSupply * (dailyRate / 100.0)

	// Create transaction for supply increase
	nonce := fmt.Sprintf("%d", now)
	tx, err := transaction.NewTransaction(
		"SYSTEM", // System address as sender
		sm.reserveAddress,
		increaseAmount,
		0, // No fee for system transactions
		transaction.SupplyIncrease,
		nonce,
		fmt.Sprintf("Daily supply increase (%.2f%%)", dailyRate),
	)
	if err != nil {
		return fmt.Errorf("failed to create supply increase transaction: %w", err)
	}

	// Submit transaction
	err = sm.txProcessor.SubmitTransaction(tx)
	if err != nil {
		return fmt.Errorf("failed to submit supply increase transaction: %w", err)
	}

	// Update total supply directly
	newSupply := currentSupply + increaseAmount
	err = sm.client.Set(sm.ctx, totalSupplyKey, newSupply, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to update total supply: %w", err)
	}

	// Update last increase timestamp
	err = sm.client.Set(sm.ctx, lastSupplyIncreaseKey, now, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to update last supply increase timestamp: %w", err)
	}

	log.Printf("Increased total supply by %.2f (%.4f%% daily rate)", increaseAmount, dailyRate)

	return nil
}

// StartScheduledTasks starts background tasks for supply management
func (sm *SupplyManager) StartScheduledTasks(ctx context.Context) {
	// Run inflation rate adjustment once per day
	go func() {
		inflationTicker := time.NewTicker(24 * time.Hour)
		defer inflationTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-inflationTicker.C:
				_, err := sm.AdjustInflationRate()
				if err != nil {
					log.Printf("Error adjusting inflation rate: %v", err)
				}
			}
		}
	}()

	// Run supply increase once per day
	go func() {
		supplyTicker := time.NewTicker(24 * time.Hour)
		defer supplyTicker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-supplyTicker.C:
				err := sm.IncreaseSupply()
				if err != nil {
					log.Printf("Error increasing supply: %v", err)
				}
			}
		}
	}()
}
