// internal/processor/transaction_processor.go
package processor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/cmatc13/stathera/internal/storage"
	"github.com/cmatc13/stathera/internal/transaction"
	"github.com/cmatc13/stathera/internal/wallet"
	"github.com/cmatc13/stathera/pkg/config"
)

var (
	// Topic for incoming transactions
	transactionTopic = "transactions"

	// Topic for processed transactions (confirmation)
	confirmedTopic = "confirmed_transactions"

	// Topic for failed transactions
	failedTopic = "failed_transactions"
)

const (
	// Consumer group ID
	consumerGroupID = "transaction_processor_group"
)

// TransactionProcessor processes incoming transactions using Kafka and Redis.
// It implements the pkg/transaction.Processor interface.
type TransactionProcessor struct {
	ctx         context.Context
	config      *config.Config
	consumer    *kafka.Reader
	producer    *kafka.Writer
	redisLedger *storage.RedisLedger
}

// NewTransactionProcessor creates a new transaction processor
func NewTransactionProcessor(ctx context.Context, cfg *config.Config) (*TransactionProcessor, error) {
	// Initialize Redis ledger
	redisLedger, err := storage.NewRedisLedger(cfg.Redis.Address)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Redis ledger: %w", err)
	}

	// Initialize Kafka consumer
	consumer := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{cfg.Kafka.Brokers},
		Topic:       transactionTopic,
		GroupID:     consumerGroupID,
		StartOffset: kafka.FirstOffset,
	})

	// Initialize Kafka producer
	producer := &kafka.Writer{
		Addr:     kafka.TCP(cfg.Kafka.Brokers),
		Balancer: &kafka.LeastBytes{},
	}

	return &TransactionProcessor{
		ctx:         ctx,
		config:      cfg,
		consumer:    consumer,
		producer:    producer,
		redisLedger: redisLedger,
	}, nil
}

// Start begins processing transactions from Kafka
func (tp *TransactionProcessor) Start() {
	// Start processing
	log.Println("Transaction processor started, waiting for transactions...")

	for {
		select {
		case <-tp.ctx.Done():
			// Context cancelled, shutdown gracefully
			log.Println("Shutting down transaction processor...")
			tp.consumer.Close()
			tp.producer.Close()
			return

		default:
			// Set a timeout for reading messages
			ctx, cancel := context.WithTimeout(tp.ctx, 100*time.Millisecond)
			// Read message
			msg, err := tp.consumer.ReadMessage(ctx)
			cancel()

			if err != nil {
				// Timeout or no message, continue
				if err == context.DeadlineExceeded {
					continue
				}
				log.Printf("Error reading message: %v", err)
				continue
			}

			// Process the transaction
			tp.processMessage(&msg)
		}
	}
}

// processMessage handles a single Kafka message containing a transaction
func (tp *TransactionProcessor) processMessage(msg *kafka.Message) {
	// Deserialize the transaction
	var tx transaction.Transaction
	if err := json.Unmarshal(msg.Value, &tx); err != nil {
		log.Printf("Error deserializing transaction: %v", err)
		// Publish to failed topic
		tp.publishFailedTransaction(&tx, fmt.Sprintf("Invalid transaction format: %v", err))
		return
	}

	log.Printf("Processing transaction: %s (Amount: %.2f, Type: %s)", tx.ID, tx.Amount, tx.Type)

	// Validate transaction signature
	if tx.Type != transaction.SupplyIncrease { // Skip signature check for system-generated transactions
		signData, err := tx.SignableData()
		if err != nil {
			tp.publishFailedTransaction(&tx, fmt.Sprintf("Failed to generate signable data: %v", err))
			return
		}

		// Get sender's public key from storage or cache
		// For simplicity, assuming public key is stored or retrievable
		publicKey, err := tp.getPublicKey(tx.Sender)
		if err != nil {
			tp.publishFailedTransaction(&tx, fmt.Sprintf("Failed to retrieve sender public key: %v", err))
			return
		}

		// Verify signature
		valid, err := wallet.VerifySignature(publicKey, signData, tx.Signature)
		if err != nil || !valid {
			tp.publishFailedTransaction(&tx, "Invalid transaction signature")
			return
		}
	}

	// Validate transaction
	if err := tx.Validate(); err != nil {
		tp.publishFailedTransaction(&tx, fmt.Sprintf("Transaction validation failed: %v", err))
		return
	}

	// Process transaction in Redis
	err := tp.redisLedger.ProcessTransaction(&tx)
	if err != nil {
		tp.publishFailedTransaction(&tx, fmt.Sprintf("Transaction processing failed: %v", err))
		return
	}

	// Update transaction status to confirmed
	tx.Status = transaction.Confirmed

	// Store confirmed transaction
	err = tp.redisLedger.StoreTransaction(&tx)
	if err != nil {
		log.Printf("Error storing transaction: %v", err)
		// Continue anyway since the transaction was processed
	}

	// Publish confirmation
	tp.publishConfirmedTransaction(&tx)

	log.Printf("Transaction %s processed successfully", tx.ID)
}

// getPublicKey retrieves a user's public key from storage
func (tp *TransactionProcessor) getPublicKey(address string) ([]byte, error) {
	// In a real implementation, you would:
	// 1. Check a cache for the public key
	// 2. If not found, retrieve from a database
	// 3. Validate that the public key generates the correct address

	// For this implementation, we'll retrieve from Redis
	pubKeyStr, err := tp.redisLedger.Client.Get(tp.ctx, "pubkey:"+address).Result()
	if err != nil {
		return nil, fmt.Errorf("public key not found for address %s", address)
	}

	return []byte(pubKeyStr), nil
}

// publishConfirmedTransaction publishes a transaction to the confirmed topic
func (tp *TransactionProcessor) publishConfirmedTransaction(tx *transaction.Transaction) {
	txJSON, err := tx.ToJSON()
	if err != nil {
		log.Printf("Error serializing confirmed transaction: %v", err)
		return
	}

	// Publish to Kafka
	err = tp.producer.WriteMessages(tp.ctx, kafka.Message{
		Topic: confirmedTopic,
		Key:   []byte(tx.ID),
		Value: txJSON,
	})

	if err != nil {
		log.Printf("Error publishing confirmed transaction: %v", err)
	}
}

// publishFailedTransaction publishes a transaction to the failed topic
func (tp *TransactionProcessor) publishFailedTransaction(tx *transaction.Transaction, reason string) {
	// Update transaction status
	tx.Status = transaction.Failed

	// Add failure reason
	tx.Description = reason

	txJSON, err := tx.ToJSON()
	if err != nil {
		log.Printf("Error serializing failed transaction: %v", err)
		return
	}

	// Publish to Kafka
	err = tp.producer.WriteMessages(tp.ctx, kafka.Message{
		Topic: failedTopic,
		Key:   []byte(tx.ID),
		Value: txJSON,
	})

	if err != nil {
		log.Printf("Error publishing failed transaction: %v", err)
	}
}

// SubmitTransaction submits a new transaction to be processed.
// This method implements the pkg/transaction.Processor interface.
func (tp *TransactionProcessor) SubmitTransaction(tx *transaction.Transaction) error {
	txJSON, err := tx.ToJSON()
	if err != nil {
		return fmt.Errorf("error serializing transaction: %w", err)
	}

	// Publish to Kafka
	err = tp.producer.WriteMessages(tp.ctx, kafka.Message{
		Topic: transactionTopic,
		Key:   []byte(tx.ID),
		Value: txJSON,
	})

	if err != nil {
		return fmt.Errorf("error publishing transaction: %w", err)
	}

	return nil
}

// GetBalance is a helper method to get a user's balance
func (tp *TransactionProcessor) GetBalance(address string) (float64, error) {
	return tp.redisLedger.GetBalance(address)
}

// SetBalance is a helper method to set a user's balance
func (tp *TransactionProcessor) SetBalance(address string, amount float64) error {
	return tp.redisLedger.SetBalance(address, amount)
}

// GetUserTransactions is a helper method to get a user's transaction history
func (tp *TransactionProcessor) GetUserTransactions(userAddress string, limit, offset int64) ([]*transaction.Transaction, error) {
	return tp.redisLedger.GetUserTransactions(userAddress, limit, offset)
}

// GetTotalSupply is a helper method to get the total supply
func (tp *TransactionProcessor) GetTotalSupply() (float64, error) {
	return tp.redisLedger.GetTotalSupply()
}

// GetInflationRate is a helper method to get the inflation rate
func (tp *TransactionProcessor) GetInflationRate() (float64, error) {
	return tp.redisLedger.GetInflationRate()
}
