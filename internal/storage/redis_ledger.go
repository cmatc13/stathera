// internal/storage/redis_ledger.go
package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cmatc13/stathera/internal/transaction"

	"github.com/go-redis/redis/v8"
)

const (
	// Balance key prefix for storing user balances
	balanceKeyPrefix = "balance:"

	// Transaction key prefix for storing transaction data
	txKeyPrefix = "tx:"

	// Queue for pending transactions
	pendingTxQueue = "queue:pending_tx"

	// Key for storing total currency supply
	totalSupplyKey = "system:total_supply"

	// Key for storing current inflation rate
	inflationRateKey = "system:inflation_rate"
)

// RedisLedger handles the storage and retrieval of account balances using Redis
type RedisLedger struct {
	Client *redis.Client
	ctx    context.Context
}

// NewRedisLedger creates a new Redis-backed ledger
func NewRedisLedger(redisAddr string) (*RedisLedger, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   0,
	})

	ctx := context.Background()

	// Test connection
	if _, err := client.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisLedger{
		Client: client,
		ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (rl *RedisLedger) Close() error {
	return rl.Client.Close()
}

// GetBalance returns the account balance for a given address
func (rl *RedisLedger) GetBalance(address string) (float64, error) {
	val, err := rl.Client.Get(rl.ctx, balanceKeyPrefix+address).Float64()
	if err == redis.Nil {
		// Address not found, return zero balance
		return 0, nil
	}
	return val, err
}

// SetBalance sets the balance for an address
func (rl *RedisLedger) SetBalance(address string, amount float64) error {
	return rl.Client.Set(rl.ctx, balanceKeyPrefix+address, amount, 0).Err()
}

// ProcessTransaction executes a transaction atomically
func (rl *RedisLedger) ProcessTransaction(tx *transaction.Transaction) error {
	// Define Lua script for atomic transaction processing
	luaScript := redis.NewScript(`
		-- Get current balances
		local senderBalance = tonumber(redis.call("GET", KEYS[1]) or "0")
		local receiverBalance = tonumber(redis.call("GET", KEYS[2]) or "0")
		local feeCollectorBalance = tonumber(redis.call("GET", KEYS[3]) or "0")
		
		-- Check transaction type
		if ARGV[4] == "PAYMENT" or ARGV[4] == "WITHDRAWAL" then
			-- Check if sender has sufficient balance
			local totalDebit = tonumber(ARGV[1]) + tonumber(ARGV[2])
			if senderBalance < totalDebit then
				return 0 -- Insufficient funds
			end
			
			-- Update balances
			redis.call("SET", KEYS[1], senderBalance - totalDebit)
			redis.call("SET", KEYS[2], receiverBalance + tonumber(ARGV[1]))
			redis.call("SET", KEYS[3], feeCollectorBalance + tonumber(ARGV[2]))
		elseif ARGV[4] == "DEPOSIT" then
			-- Update receiver balance and fee collector
			redis.call("SET", KEYS[2], receiverBalance + tonumber(ARGV[1]))
			redis.call("SET", KEYS[3], feeCollectorBalance + tonumber(ARGV[2]))
		elseif ARGV[4] == "SUPPLY_INCREASE" then
			-- Update total supply and receiver balance
			local totalSupply = tonumber(redis.call("GET", KEYS[4]) or "0")
			redis.call("SET", KEYS[4], totalSupply + tonumber(ARGV[1]))
			redis.call("SET", KEYS[2], receiverBalance + tonumber(ARGV[1]))
		end
		
		-- Store transaction data
		redis.call("SET", KEYS[5], ARGV[3])
		return 1
	`)

	// Prepare keys and arguments
	senderKey := balanceKeyPrefix + tx.Sender
	receiverKey := balanceKeyPrefix + tx.Receiver
	feeCollectorKey := balanceKeyPrefix + "fee_collector" // System fee collector address
	totalSupplyKey := "system:total_supply"
	txKey := txKeyPrefix + tx.ID

	// Serialize transaction to JSON
	txJSON, err := tx.ToJSON()
	if err != nil {
		return err
	}

	// Execute Lua script
	res, err := luaScript.Run(rl.ctx, rl.Client,
		[]string{senderKey, receiverKey, feeCollectorKey, totalSupplyKey, txKey},
		tx.Amount, tx.Fee, string(txJSON), string(tx.Type)).Result()

	if err != nil {
		return fmt.Errorf("failed to execute transaction: %w", err)
	}

	if res.(int64) == 0 {
		return errors.New("insufficient funds")
	}

	return nil
}

// StoreTransaction stores a transaction in Redis
func (rl *RedisLedger) StoreTransaction(tx *transaction.Transaction) error {
	txJSON, err := tx.ToJSON()
	if err != nil {
		return err
	}

	// Store by ID
	err = rl.Client.Set(rl.ctx, txKeyPrefix+tx.ID, txJSON, 0).Err()
	if err != nil {
		return err
	}

	// Add to user's transaction history (using sorted sets)
	err = rl.Client.ZAdd(rl.ctx, "user:"+tx.Sender+":txs", &redis.Z{
		Score:  float64(tx.Timestamp),
		Member: tx.ID,
	}).Err()
	if err != nil {
		return err
	}

	if tx.Sender != tx.Receiver {
		err = rl.Client.ZAdd(rl.ctx, "user:"+tx.Receiver+":txs", &redis.Z{
			Score:  float64(tx.Timestamp),
			Member: tx.ID,
		}).Err()
		if err != nil {
			return err
		}
	}

	return nil
}

// QueueTransaction adds a transaction to the processing queue
func (rl *RedisLedger) QueueTransaction(tx *transaction.Transaction) error {
	txJSON, err := tx.ToJSON()
	if err != nil {
		return err
	}

	return rl.Client.LPush(rl.ctx, pendingTxQueue, txJSON).Err()
}

// GetPendingTransaction retrieves a transaction from the queue
func (rl *RedisLedger) GetPendingTransaction() (*transaction.Transaction, error) {
	result, err := rl.Client.BRPop(rl.ctx, 0, pendingTxQueue).Result()
	if err != nil {
		return nil, err
	}

	// result[0] is the queue name, result[1] is the value
	txJSON := result[1]
	return transaction.FromJSON([]byte(txJSON))
}

// GetTransaction retrieves a transaction by ID
func (rl *RedisLedger) GetTransaction(txID string) (*transaction.Transaction, error) {
	txJSON, err := rl.Client.Get(rl.ctx, txKeyPrefix+txID).Result()
	if err == redis.Nil {
		return nil, fmt.Errorf("transaction not found: %s", txID)
	}
	if err != nil {
		return nil, err
	}

	return transaction.FromJSON([]byte(txJSON))
}

// GetUserTransactions retrieves a user's transaction history
func (rl *RedisLedger) GetUserTransactions(userAddress string, limit, offset int64) ([]*transaction.Transaction, error) {
	// Get transaction IDs from sorted set
	txIDs, err := rl.Client.ZRevRange(rl.ctx, "user:"+userAddress+":txs", offset, offset+limit-1).Result()
	if err != nil {
		return nil, err
	}

	transactions := make([]*transaction.Transaction, 0, len(txIDs))
	for _, txID := range txIDs {
		tx, err := rl.GetTransaction(txID)
		if err != nil {
			return nil, err
		}
		transactions = append(transactions, tx)
	}

	return transactions, nil
}

// SetTotalSupply sets the current total supply of the currency
func (rl *RedisLedger) SetTotalSupply(amount float64) error {
	return rl.Client.Set(rl.ctx, totalSupplyKey, amount, 0).Err()
}

// GetTotalSupply gets the current total supply of the currency
func (rl *RedisLedger) GetTotalSupply() (float64, error) {
	return rl.Client.Get(rl.ctx, totalSupplyKey).Float64()
}

// SetInflationRate sets the current inflation rate
func (rl *RedisLedger) SetInflationRate(rate float64) error {
	return rl.Client.Set(rl.ctx, inflationRateKey, rate, 0).Err()
}

// GetInflationRate gets the current inflation rate
func (rl *RedisLedger) GetInflationRate() (float64, error) {
	return rl.Client.Get(rl.ctx, inflationRateKey).Float64()
}

// ProcessSupplyInflation executes the annual inflation process
func (rl *RedisLedger) ProcessSupplyInflation(currentInflationRate float64, destinationAddress string) error {
	// Get current total supply
	totalSupply, err := rl.GetTotalSupply()
	if err != nil {
		return err
	}

	// Calculate new coins to mint
	newCoins := totalSupply * (currentInflationRate / 100.0)

	// Create supply increase transaction
	tx, err := transaction.NewTransaction(
		"SYSTEM", // System address as sender
		destinationAddress,
		newCoins,
		0,
		transaction.SupplyIncrease,
		fmt.Sprintf("%d", time.Now().Unix()),
		"Annual supply inflation",
	)
	if err != nil {
		return err
	}

	// Process the transaction
	err = rl.ProcessTransaction(tx)
	if err != nil {
		return err
	}

	// Update total supply
	newTotalSupply := totalSupply + newCoins
	return rl.SetTotalSupply(newTotalSupply)
}
