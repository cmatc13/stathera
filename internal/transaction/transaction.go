// internal/transaction/transaction.go
package transaction

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TransactionType defines the type of transaction
type TransactionType string

const (
	// Payment from one user to another
	Payment TransactionType = "PAYMENT"
	// Deposit from external source
	Deposit TransactionType = "DEPOSIT"
	// Withdrawal to external destination
	Withdrawal TransactionType = "WITHDRAWAL"
	// Fee transaction
	Fee TransactionType = "FEE"
	// SupplyIncrease represents new coins from inflation
	SupplyIncrease TransactionType = "SUPPLY_INCREASE"
)

// TransactionStatus defines the status of a transaction
type TransactionStatus string

const (
	// Pending transactions are being processed
	Pending TransactionStatus = "PENDING"
	// Confirmed transactions have been validated and applied
	Confirmed TransactionStatus = "CONFIRMED"
	// Failed transactions were not successful
	Failed TransactionStatus = "FAILED"
)

// Transaction represents a transfer of funds between addresses
type Transaction struct {
	ID          string            `json:"id"`
	Sender      string            `json:"sender"`
	Receiver    string            `json:"receiver"`
	Amount      float64           `json:"amount"`
	Fee         float64           `json:"fee"`
	Type        TransactionType   `json:"type"`
	Status      TransactionStatus `json:"status"`
	Nonce       string            `json:"nonce"`
	Signature   []byte            `json:"signature"`
	Timestamp   int64             `json:"timestamp"`
	Description string            `json:"description,omitempty"`
	Hash        string            `json:"hash"`
}

// NewTransaction creates a new transaction without signature
func NewTransaction(sender, receiver string, amount, fee float64, txType TransactionType, nonce string, description string) (*Transaction, error) {
	if amount <= 0 {
		return nil, fmt.Errorf("amount must be positive")
	}

	tx := &Transaction{
		ID:          uuid.New().String(),
		Sender:      sender,
		Receiver:    receiver,
		Amount:      amount,
		Fee:         fee,
		Type:        txType,
		Status:      Pending,
		Nonce:       nonce,
		Timestamp:   time.Now().Unix(),
		Description: description,
	}

	// Calculate transaction hash
	hash, err := tx.CalculateHash()
	if err != nil {
		return nil, err
	}
	tx.Hash = hash

	return tx, nil
}

// SignableData returns the data that should be signed
func (tx *Transaction) SignableData() ([]byte, error) {
	// Create a composite string of transaction data
	signData := fmt.Sprintf("%s|%s|%s|%.8f|%.8f|%s|%s|%d",
		tx.ID, tx.Sender, tx.Receiver, tx.Amount, tx.Fee, tx.Type, tx.Nonce, tx.Timestamp)
	return []byte(signData), nil
}

// CalculateHash calculates the transaction hash
func (tx *Transaction) CalculateHash() (string, error) {
	// Serialize transaction (without signature and hash)
	txCopy := *tx
	txCopy.Signature = nil
	txCopy.Hash = ""

	txJSON, err := json.Marshal(txCopy)
	if err != nil {
		return "", fmt.Errorf("failed to serialize transaction: %w", err)
	}

	// Calculate SHA256 hash
	h := sha256.New()
	h.Write(txJSON)
	hash := h.Sum(nil)

	return hex.EncodeToString(hash), nil
}

// Validate checks if the transaction is valid
func (tx *Transaction) Validate() error {
	// Basic validation
	if tx.Amount <= 0 {
		return fmt.Errorf("transaction amount must be positive")
	}

	if tx.Sender == tx.Receiver && tx.Type == Payment {
		return fmt.Errorf("sender and receiver cannot be the same for payment transactions")
	}

	// Validate hash
	calculatedHash, err := tx.CalculateHash()
	if err != nil {
		return err
	}

	if calculatedHash != tx.Hash {
		return fmt.Errorf("transaction hash is invalid")
	}

	return nil
}

// ToJSON serializes the transaction to JSON
func (tx *Transaction) ToJSON() ([]byte, error) {
	return json.Marshal(tx)
}

// FromJSON deserializes the transaction from JSON
func FromJSON(data []byte) (*Transaction, error) {
	var tx Transaction
	if err := json.Unmarshal(data, &tx); err != nil {
		return nil, fmt.Errorf("failed to deserialize transaction: %w", err)
	}
	return &tx, nil
}
