// Package transaction implements the transaction engine (middle layer) of the Stathera system.
// It provides high-speed transaction processing, signature validation, and account management.
package transaction

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cmatc13/stathera/timeoracle"
)

// Common errors
var (
	ErrInvalidAmount      = errors.New("invalid amount")
	ErrInvalidSignature   = errors.New("invalid signature")
	ErrInsufficientFunds  = errors.New("insufficient funds")
	ErrInvalidTransaction = errors.New("invalid transaction")
	ErrDuplicateNonce     = errors.New("duplicate nonce")
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
	// Settled transactions have been finalized in the canonical ledger
	Settled TransactionStatus = "SETTLED"
)

// Transaction represents a transfer of funds between addresses
type Transaction struct {
	ID          string                `json:"id"`
	Sender      string                `json:"sender"`
	Receiver    string                `json:"receiver"`
	Amount      float64               `json:"amount"`
	Fee         float64               `json:"fee"`
	Type        TransactionType       `json:"type"`
	Status      TransactionStatus     `json:"status"`
	Nonce       string                `json:"nonce"`
	Signature   []byte                `json:"signature"`
	Timestamp   int64                 `json:"timestamp"`
	TimeProof   *timeoracle.TimeProof `json:"time_proof,omitempty"`
	Description string                `json:"description,omitempty"`
	Hash        string                `json:"hash"`
}

// NewTransaction creates a new transaction without signature
func NewTransaction(sender, receiver string, amount, fee float64, txType TransactionType, nonce string, description string) (*Transaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	tx := &Transaction{
		ID:          generateID(),
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
	// Create a composite string of transaction data (without signature and hash)
	hashData := fmt.Sprintf("%s|%s|%s|%.8f|%.8f|%s|%s|%d|%s",
		tx.ID, tx.Sender, tx.Receiver, tx.Amount, tx.Fee, tx.Type, tx.Nonce, tx.Timestamp, tx.Description)

	// Calculate SHA256 hash
	h := sha256.Sum256([]byte(hashData))
	return hex.EncodeToString(h[:]), nil
}

// Sign signs the transaction with the provided private key
func (tx *Transaction) Sign(privateKey ed25519.PrivateKey) error {
	signData, err := tx.SignableData()
	if err != nil {
		return err
	}

	tx.Signature = ed25519.Sign(privateKey, signData)
	return nil
}

// Verify checks if the transaction signature is valid
func (tx *Transaction) Verify(publicKey ed25519.PublicKey) (bool, error) {
	if len(tx.Signature) == 0 {
		return false, ErrInvalidSignature
	}

	signData, err := tx.SignableData()
	if err != nil {
		return false, err
	}

	return ed25519.Verify(publicKey, signData, tx.Signature), nil
}

// Validate checks if the transaction is valid
func (tx *Transaction) Validate() error {
	// Basic validation
	if tx.Amount <= 0 {
		return ErrInvalidAmount
	}

	if tx.Sender == tx.Receiver && tx.Type == Payment {
		return errors.New("sender and receiver cannot be the same for payment transactions")
	}

	// Validate hash
	calculatedHash, err := tx.CalculateHash()
	if err != nil {
		return err
	}

	if calculatedHash != tx.Hash {
		return errors.New("transaction hash is invalid")
	}

	return nil
}

// Account represents a user account in the system
type Account struct {
	Address    string            `json:"address"`
	Balance    float64           `json:"balance"`
	PublicKey  ed25519.PublicKey `json:"public_key"`
	Nonces     map[string]bool   `json:"nonces"`
	LastActive int64             `json:"last_active"`
}

// NewAccount creates a new account
func NewAccount(address string, publicKey ed25519.PublicKey) *Account {
	return &Account{
		Address:    address,
		Balance:    0,
		PublicKey:  publicKey,
		Nonces:     make(map[string]bool),
		LastActive: time.Now().Unix(),
	}
}

// TransactionEngine manages accounts and processes transactions
type TransactionEngine struct {
	mu           sync.RWMutex
	accounts     map[string]*Account
	transactions map[string]*Transaction
	timeOracle   timeoracle.TimeOracle
	feeAddress   string
}

// NewTransactionEngine creates a new transaction engine
func NewTransactionEngine(timeOracle timeoracle.TimeOracle, feeAddress string) *TransactionEngine {
	return &TransactionEngine{
		accounts:     make(map[string]*Account),
		transactions: make(map[string]*Transaction),
		timeOracle:   timeOracle,
		feeAddress:   feeAddress,
	}
}

// CreateAccount creates a new account
func (e *TransactionEngine) CreateAccount(address string, publicKey ed25519.PublicKey) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.accounts[address]; exists {
		return fmt.Errorf("account %s already exists", address)
	}

	e.accounts[address] = NewAccount(address, publicKey)
	return nil
}

// GetAccount returns an account by address
func (e *TransactionEngine) GetAccount(address string) (*Account, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	account, exists := e.accounts[address]
	if !exists {
		return nil, fmt.Errorf("account %s not found", address)
	}

	return account, nil
}

// GetBalance returns the balance of an account
func (e *TransactionEngine) GetBalance(address string) (float64, error) {
	account, err := e.GetAccount(address)
	if err != nil {
		return 0, err
	}

	return account.Balance, nil
}

// ProcessTransaction processes a transaction
func (e *TransactionEngine) ProcessTransaction(tx *Transaction) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Check if transaction already exists
	if _, exists := e.transactions[tx.ID]; exists {
		return fmt.Errorf("transaction %s already exists", tx.ID)
	}

	// Validate transaction
	if err := tx.Validate(); err != nil {
		tx.Status = Failed
		e.transactions[tx.ID] = tx
		return err
	}

	// Skip signature check for system transactions
	if tx.Type != SupplyIncrease {
		// Get sender account
		sender, exists := e.accounts[tx.Sender]
		if !exists {
			tx.Status = Failed
			e.transactions[tx.ID] = tx
			return fmt.Errorf("sender account %s not found", tx.Sender)
		}

		// Check for duplicate nonce
		if sender.Nonces[tx.Nonce] {
			tx.Status = Failed
			e.transactions[tx.ID] = tx
			return ErrDuplicateNonce
		}

		// Verify signature
		valid, err := tx.Verify(sender.PublicKey)
		if err != nil || !valid {
			tx.Status = Failed
			e.transactions[tx.ID] = tx
			return ErrInvalidSignature
		}

		// Check sufficient funds for payments and withdrawals
		if tx.Type == Payment || tx.Type == Withdrawal {
			if sender.Balance < tx.Amount+tx.Fee {
				tx.Status = Failed
				e.transactions[tx.ID] = tx
				return ErrInsufficientFunds
			}
		}
	}

	// Process transaction based on type
	switch tx.Type {
	case Payment:
		// Get receiver account
		receiver, exists := e.accounts[tx.Receiver]
		if !exists {
			tx.Status = Failed
			e.transactions[tx.ID] = tx
			return fmt.Errorf("receiver account %s not found", tx.Receiver)
		}

		// Update balances
		sender := e.accounts[tx.Sender]
		sender.Balance -= tx.Amount + tx.Fee
		receiver.Balance += tx.Amount

		// Update fee account
		if tx.Fee > 0 {
			feeAccount, exists := e.accounts[e.feeAddress]
			if exists {
				feeAccount.Balance += tx.Fee
			}
		}

		// Record nonce
		sender.Nonces[tx.Nonce] = true
		sender.LastActive = tx.Timestamp
		receiver.LastActive = tx.Timestamp

	case Deposit:
		// Get receiver account
		receiver, exists := e.accounts[tx.Receiver]
		if !exists {
			tx.Status = Failed
			e.transactions[tx.ID] = tx
			return fmt.Errorf("receiver account %s not found", tx.Receiver)
		}

		// Update balance
		receiver.Balance += tx.Amount

		// Update fee account
		if tx.Fee > 0 {
			feeAccount, exists := e.accounts[e.feeAddress]
			if exists {
				feeAccount.Balance += tx.Fee
			}
		}

		receiver.LastActive = tx.Timestamp

	case Withdrawal:
		// Update sender balance
		sender := e.accounts[tx.Sender]
		sender.Balance -= tx.Amount + tx.Fee

		// Update fee account
		if tx.Fee > 0 {
			feeAccount, exists := e.accounts[e.feeAddress]
			if exists {
				feeAccount.Balance += tx.Fee
			}
		}

		// Record nonce
		sender.Nonces[tx.Nonce] = true
		sender.LastActive = tx.Timestamp

	case SupplyIncrease:
		// Get receiver account (reserve)
		receiver, exists := e.accounts[tx.Receiver]
		if !exists {
			tx.Status = Failed
			e.transactions[tx.ID] = tx
			return fmt.Errorf("reserve account %s not found", tx.Receiver)
		}

		// Update balance
		receiver.Balance += tx.Amount
		receiver.LastActive = tx.Timestamp
	}

	// Update transaction status
	tx.Status = Confirmed

	// Store transaction
	e.transactions[tx.ID] = tx

	return nil
}

// GetTransaction returns a transaction by ID
func (e *TransactionEngine) GetTransaction(id string) (*Transaction, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	tx, exists := e.transactions[id]
	if !exists {
		return nil, fmt.Errorf("transaction %s not found", id)
	}

	return tx, nil
}

// GetTransactions returns all transactions
func (e *TransactionEngine) GetTransactions() []*Transaction {
	e.mu.RLock()
	defer e.mu.RUnlock()

	txs := make([]*Transaction, 0, len(e.transactions))
	for _, tx := range e.transactions {
		txs = append(txs, tx)
	}

	return txs
}

// GetPendingTransactions returns all pending transactions
func (e *TransactionEngine) GetPendingTransactions() []*Transaction {
	e.mu.RLock()
	defer e.mu.RUnlock()

	txs := make([]*Transaction, 0)
	for _, tx := range e.transactions {
		if tx.Status == Pending {
			txs = append(txs, tx)
		}
	}

	return txs
}

// GetConfirmedTransactions returns all confirmed transactions
func (e *TransactionEngine) GetConfirmedTransactions() []*Transaction {
	e.mu.RLock()
	defer e.mu.RUnlock()

	txs := make([]*Transaction, 0)
	for _, tx := range e.transactions {
		if tx.Status == Confirmed {
			txs = append(txs, tx)
		}
	}

	return txs
}

// MarkTransactionsAsSettled marks transactions as settled
func (e *TransactionEngine) MarkTransactionsAsSettled(txIDs []string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, id := range txIDs {
		tx, exists := e.transactions[id]
		if !exists {
			return fmt.Errorf("transaction %s not found", id)
		}

		if tx.Status != Confirmed {
			return fmt.Errorf("transaction %s is not confirmed", id)
		}

		tx.Status = Settled
	}

	return nil
}

// generateID generates a unique transaction ID
func generateID() string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(hash[:])[:16]
}
