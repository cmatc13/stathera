// Package settlement implements the settlement layer (top layer) of the Stathera system.
// It provides batch transaction finality, cryptographic proofs, and ensures
// eventual consistency between the high-speed transaction layer and the canonical ledger.
package settlement

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/cmatc13/stathera/timeoracle"
	"github.com/cmatc13/stathera/transaction"
)

// Common errors
var (
	ErrEmptyBatch        = errors.New("empty batch")
	ErrInvalidMerkleRoot = errors.New("invalid merkle root")
	ErrSettlementFailed  = errors.New("settlement failed")
)

// SettlementBatch represents a batch of transactions to be settled
type SettlementBatch struct {
	ID            string                `json:"id"`
	Transactions  []string              `json:"transactions"` // Transaction IDs
	MerkleRoot    string                `json:"merkle_root"`
	Timestamp     int64                 `json:"timestamp"`
	TimeProof     *timeoracle.TimeProof `json:"time_proof"`
	PrevBatchID   string                `json:"prev_batch_id"`
	LedgerEntryID string                `json:"ledger_entry_id"`
	Status        string                `json:"status"`
}

// SettlementEngine handles the settlement of transactions to the canonical ledger
type SettlementEngine struct {
	mu              sync.RWMutex
	batches         map[string]*SettlementBatch
	txEngine        TransactionProcessor
	canonicalLedger LedgerManager
	timeOracle      timeoracle.TimeOracle
	batchSize       int
	settleInterval  time.Duration
	latestBatchID   string
}

// TransactionProcessor defines the interface for the transaction layer
type TransactionProcessor interface {
	// GetConfirmedTransactions returns all confirmed transactions
	GetConfirmedTransactions() []*transaction.Transaction

	// MarkTransactionsAsSettled marks transactions as settled
	MarkTransactionsAsSettled(txIDs []string) error

	// GetTransaction returns a transaction by ID
	GetTransaction(id string) (*transaction.Transaction, error)
}

// LedgerManager defines the interface for the ledger layer
type LedgerManager interface {
	// GetTotalSupply returns the current total supply
	GetTotalSupply() float64

	// GetLatestHash returns the hash of the latest ledger entry
	GetLatestHash() string

	// MintSupply increases the total supply based on the provided inflation rate
	MintSupply(ctx context.Context, inflationRate float64, reason string) error

	// VerifyIntegrity checks the integrity of the entire ledger chain
	VerifyIntegrity() (bool, error)
}

// NewSettlementEngine creates a new settlement engine
func NewSettlementEngine(
	txEngine TransactionProcessor,
	canonicalLedger LedgerManager,
	timeOracle timeoracle.TimeOracle,
	batchSize int,
	settleInterval time.Duration,
) *SettlementEngine {
	return &SettlementEngine{
		batches:         make(map[string]*SettlementBatch),
		txEngine:        txEngine,
		canonicalLedger: canonicalLedger,
		timeOracle:      timeOracle,
		batchSize:       batchSize,
		settleInterval:  settleInterval,
		latestBatchID:   "",
	}
}

// StartSettlementProcess starts the periodic settlement process
func (e *SettlementEngine) StartSettlementProcess(ctx context.Context) {
	ticker := time.NewTicker(e.settleInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := e.SettleTransactions(ctx); err != nil {
				// Log error but continue
				fmt.Printf("Settlement error: %v\n", err)
			}
		}
	}
}

// SettleTransactions creates a batch of transactions and settles them to the ledger
func (e *SettlementEngine) SettleTransactions(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Get confirmed transactions
	confirmedTxs := e.txEngine.GetConfirmedTransactions()
	if len(confirmedTxs) == 0 {
		return ErrEmptyBatch
	}

	// Limit batch size
	batchSize := e.batchSize
	if batchSize > len(confirmedTxs) {
		batchSize = len(confirmedTxs)
	}

	// Select transactions for this batch
	selectedTxs := confirmedTxs[:batchSize]

	// Extract transaction IDs
	txIDs := make([]string, len(selectedTxs))
	for i, tx := range selectedTxs {
		txIDs[i] = tx.ID
	}

	// Create merkle tree
	merkleRoot, err := e.calculateMerkleRoot(txIDs)
	if err != nil {
		return err
	}

	// Get time with proof
	timestamp, timeProof, err := e.timeOracle.GetTimeWithProof()
	if err != nil {
		return err
	}

	// Create batch
	batch := &SettlementBatch{
		ID:           generateID(),
		Transactions: txIDs,
		MerkleRoot:   merkleRoot,
		Timestamp:    timestamp,
		TimeProof:    timeProof,
		PrevBatchID:  e.latestBatchID,
		Status:       "PENDING",
	}

	// Store batch
	e.batches[batch.ID] = batch

	// Update latest batch ID
	e.latestBatchID = batch.ID

	// Mark transactions as settled
	if err := e.txEngine.MarkTransactionsAsSettled(txIDs); err != nil {
		batch.Status = "FAILED"
		return err
	}

	// Update batch status
	batch.Status = "SETTLED"

	return nil
}

// calculateMerkleRoot calculates the merkle root of a list of transaction IDs
func (e *SettlementEngine) calculateMerkleRoot(txIDs []string) (string, error) {
	if len(txIDs) == 0 {
		return "", ErrEmptyBatch
	}

	// For a simple implementation, just hash all transaction IDs together
	// In a production system, you would build a proper merkle tree
	h := sha256.New()
	for _, id := range txIDs {
		h.Write([]byte(id))
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// GetBatch returns a settlement batch by ID
func (e *SettlementEngine) GetBatch(id string) (*SettlementBatch, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	batch, exists := e.batches[id]
	if !exists {
		return nil, fmt.Errorf("batch %s not found", id)
	}

	return batch, nil
}

// GetLatestBatch returns the latest settlement batch
func (e *SettlementEngine) GetLatestBatch() (*SettlementBatch, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if e.latestBatchID == "" {
		return nil, errors.New("no batches available")
	}

	return e.batches[e.latestBatchID], nil
}

// VerifyBatch verifies the integrity of a settlement batch
func (e *SettlementEngine) VerifyBatch(batch *SettlementBatch) error {
	if batch == nil {
		return errors.New("batch cannot be nil")
	}

	// Verify time proof
	if err := e.timeOracle.VerifyProof(batch.TimeProof); err != nil {
		return err
	}

	// Verify merkle root
	calculatedRoot, err := e.calculateMerkleRoot(batch.Transactions)
	if err != nil {
		return err
	}

	if calculatedRoot != batch.MerkleRoot {
		return ErrInvalidMerkleRoot
	}

	return nil
}

// generateID generates a unique batch ID
func generateID() string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
	return hex.EncodeToString(h.Sum(nil))[:16]
}
