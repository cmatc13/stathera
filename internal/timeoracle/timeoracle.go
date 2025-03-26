// Package timeoracle implements a secure, self-contained time governance system
// that provides cryptographic time proofs for the Stathera monetary system.
package timeoracle

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Common errors
var (
	ErrInvalidTimestamp = errors.New("invalid timestamp")
	ErrInvalidProof     = errors.New("invalid time proof")
	ErrFutureTimestamp  = errors.New("timestamp is in the future")
	ErrExpiredProof     = errors.New("time proof has expired")
)

// TimeOracle defines the interface for time-related operations
type TimeOracle interface {
	// Now returns the current timestamp
	Now() int64

	// Validate checks if a timestamp is valid
	Validate(timestamp int64) error

	// GenerateProof creates a cryptographic time proof for the current time
	GenerateProof() (*TimeProof, error)

	// VerifyProof checks if a time proof is valid
	VerifyProof(proof *TimeProof) error

	// GetTimeWithProof returns the current time with a cryptographic proof
	GetTimeWithProof() (int64, *TimeProof, error)
}

// TimeProof represents a cryptographic proof of time
type TimeProof struct {
	Timestamp int64  `json:"timestamp"`
	Nonce     uint64 `json:"nonce"`
	Signature []byte `json:"signature"`
}

// StandardTimeOracle implements a secure time oracle using HMAC-SHA256
type StandardTimeOracle struct {
	mu            sync.RWMutex
	secret        []byte
	maxDrift      time.Duration
	proofValidity time.Duration
	proofCache    map[int64]TimeProof
}

// NewStandardTimeOracle creates a new standard time oracle
func NewStandardTimeOracle(secret []byte, maxDrift, proofValidity time.Duration) (*StandardTimeOracle, error) {
	if len(secret) < 32 {
		return nil, errors.New("secret must be at least 32 bytes")
	}

	return &StandardTimeOracle{
		secret:        secret,
		maxDrift:      maxDrift,
		proofValidity: proofValidity,
		proofCache:    make(map[int64]TimeProof),
	}, nil
}

// Now returns the current timestamp
func (o *StandardTimeOracle) Now() int64 {
	return time.Now().Unix()
}

// Validate checks if a timestamp is valid
func (o *StandardTimeOracle) Validate(timestamp int64) error {
	now := time.Now().Unix()

	// Check if timestamp is in the future (with allowed drift)
	maxAllowed := now + int64(o.maxDrift.Seconds())
	if timestamp > maxAllowed {
		return fmt.Errorf("%w: timestamp %d is beyond max allowed %d",
			ErrFutureTimestamp, timestamp, maxAllowed)
	}

	// Check if timestamp is too old
	minAllowed := now - int64(o.proofValidity.Seconds())
	if timestamp < minAllowed {
		return fmt.Errorf("%w: timestamp %d is before min allowed %d",
			ErrExpiredProof, timestamp, minAllowed)
	}

	return nil
}

// GenerateProof creates a cryptographic time proof for the current time
func (o *StandardTimeOracle) GenerateProof() (*TimeProof, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	// Get current time
	now := time.Now().Unix()

	// Check if we have a cached proof for this second
	if proof, exists := o.proofCache[now]; exists {
		return &proof, nil
	}

	// Generate a new proof
	nonce := uint64(time.Now().UnixNano())

	// Create signature
	signature, err := o.signTimestamp(now, nonce)
	if err != nil {
		return nil, err
	}

	proof := TimeProof{
		Timestamp: now,
		Nonce:     nonce,
		Signature: signature,
	}

	// Cache the proof
	o.proofCache[now] = proof

	// Clean old proofs from cache
	o.cleanCache()

	return &proof, nil
}

// VerifyProof checks if a time proof is valid
func (o *StandardTimeOracle) VerifyProof(proof *TimeProof) error {
	if proof == nil {
		return errors.New("proof cannot be nil")
	}

	// Validate timestamp
	if err := o.Validate(proof.Timestamp); err != nil {
		return err
	}

	// Verify signature
	expectedSignature, err := o.signTimestamp(proof.Timestamp, proof.Nonce)
	if err != nil {
		return err
	}

	if !hmac.Equal(proof.Signature, expectedSignature) {
		return ErrInvalidProof
	}

	return nil
}

// signTimestamp creates an HMAC-SHA256 signature for a timestamp and nonce
func (o *StandardTimeOracle) signTimestamp(timestamp int64, nonce uint64) ([]byte, error) {
	h := hmac.New(sha256.New, o.secret)

	// Write timestamp
	err := binary.Write(h, binary.BigEndian, timestamp)
	if err != nil {
		return nil, err
	}

	// Write nonce
	err = binary.Write(h, binary.BigEndian, nonce)
	if err != nil {
		return nil, err
	}

	return h.Sum(nil), nil
}

// cleanCache removes expired proofs from the cache
func (o *StandardTimeOracle) cleanCache() {
	now := time.Now().Unix()
	minAllowed := now - int64(o.proofValidity.Seconds())

	for ts := range o.proofCache {
		if ts < minAllowed {
			delete(o.proofCache, ts)
		}
	}
}

// GetTimeWithProof returns the current time with a cryptographic proof
func (o *StandardTimeOracle) GetTimeWithProof() (int64, *TimeProof, error) {
	proof, err := o.GenerateProof()
	if err != nil {
		return 0, nil, err
	}

	return proof.Timestamp, proof, nil
}
