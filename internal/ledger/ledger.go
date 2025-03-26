// Package ledger implements the canonical ledger (base layer) of the Stathera system.
// It provides immutable, cryptographically secure record-keeping and
// implements deterministic minting with simple annual issuance.
package ledger

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
)

// Common errors
var (
	ErrInvalidAmount      = errors.New("invalid amount")
	ErrInvalidSupplyDelta = errors.New("invalid supply delta")
	ErrInvalidTimestamp   = errors.New("invalid timestamp")
)

// LedgerEntry represents an immutable entry in the canonical ledger
type LedgerEntry struct {
	Timestamp   int64   `json:"timestamp"`
	TotalSupply float64 `json:"total_supply"`
	Delta       float64 `json:"delta"`
	Reason      string  `json:"reason"`
	Hash        string  `json:"hash"`
	PrevHash    string  `json:"prev_hash"`
}

// CalculateHash computes the SHA-256 hash of the ledger entry
func (e *LedgerEntry) CalculateHash() string {
	data := fmt.Sprintf("%d|%.8f|%.8f|%s|%s",
		e.Timestamp, e.TotalSupply, e.Delta, e.Reason, e.PrevHash)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// Ledger represents the canonical ledger for the monetary system
type Ledger struct {
	mu           sync.RWMutex
	totalSupply  float64
	entries      []*LedgerEntry
	latestHash   string
	minInflation float64
	maxInflation float64
	timeOracle   TimeOracle
}

// TimeOracle defines the interface for time-related operations
type TimeOracle interface {
	// Now returns the current timestamp
	Now() int64

	// Validate checks if a timestamp is valid
	Validate(timestamp int64) error
}

// NewLedger creates a new canonical ledger
func NewLedger(initialSupply, minInflation, maxInflation float64, timeOracle TimeOracle) (*Ledger, error) {
	if initialSupply <= 0 {
		return nil, ErrInvalidAmount
	}

	if minInflation < 0 || maxInflation <= minInflation {
		return nil, ErrInvalidSupplyDelta
	}

	if timeOracle == nil {
		return nil, errors.New("time oracle cannot be nil")
	}

	l := &Ledger{
		totalSupply:  initialSupply,
		entries:      make([]*LedgerEntry, 0),
		latestHash:   "",
		minInflation: minInflation,
		maxInflation: maxInflation,
		timeOracle:   timeOracle,
	}

	// Create genesis entry
	now := timeOracle.Now()
	entry := &LedgerEntry{
		Timestamp:   now,
		TotalSupply: initialSupply,
		Delta:       initialSupply,
		Reason:      "Genesis",
		PrevHash:    "",
	}

	entry.Hash = entry.CalculateHash()
	l.entries = append(l.entries, entry)
	l.latestHash = entry.Hash

	return l, nil
}

// GetTotalSupply returns the current total supply
func (l *Ledger) GetTotalSupply() float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.totalSupply
}

// GetEntries returns all ledger entries
func (l *Ledger) GetEntries() []*LedgerEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	// Return a copy to prevent modification
	entries := make([]*LedgerEntry, len(l.entries))
	copy(entries, l.entries)
	return entries
}

// GetLatestHash returns the hash of the latest ledger entry
func (l *Ledger) GetLatestHash() string {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.latestHash
}

// MintSupply increases the total supply based on the provided inflation rate
// This is the only way to increase the total supply in the system
func (l *Ledger) MintSupply(ctx context.Context, inflationRate float64, reason string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Validate inflation rate
	if inflationRate < l.minInflation || inflationRate > l.maxInflation {
		return fmt.Errorf("inflation rate %.4f outside allowed range [%.4f, %.4f]",
			inflationRate, l.minInflation, l.maxInflation)
	}

	// Get current timestamp
	now := l.timeOracle.Now()

	// Calculate supply increase
	delta := l.totalSupply * (inflationRate / 100.0)
	newSupply := l.totalSupply + delta

	// Create new ledger entry
	entry := &LedgerEntry{
		Timestamp:   now,
		TotalSupply: newSupply,
		Delta:       delta,
		Reason:      reason,
		PrevHash:    l.latestHash,
	}

	// Calculate hash
	entry.Hash = entry.CalculateHash()

	// Update ledger state
	l.totalSupply = newSupply
	l.entries = append(l.entries, entry)
	l.latestHash = entry.Hash

	return nil
}

// VerifyIntegrity checks the integrity of the entire ledger chain
func (l *Ledger) VerifyIntegrity() (bool, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if len(l.entries) == 0 {
		return false, errors.New("ledger is empty")
	}

	// Verify each entry
	for i, entry := range l.entries {
		// Recalculate hash
		calculatedHash := entry.CalculateHash()
		if calculatedHash != entry.Hash {
			return false, fmt.Errorf("invalid hash at entry %d", i)
		}

		// Verify chain (except for genesis)
		if i > 0 {
			if entry.PrevHash != l.entries[i-1].Hash {
				return false, fmt.Errorf("broken chain at entry %d", i)
			}
		}
	}

	return true, nil
}
