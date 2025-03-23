// Package transaction provides interfaces and types for transaction processing.
package transaction

import (
	"github.com/cmatc13/stathera/internal/transaction"
)

// Processor defines the interface for submitting transactions.
// This interface is used by components that need to submit transactions
// without directly depending on the transaction processor implementation.
type Processor interface {
	// SubmitTransaction submits a new transaction to be processed.
	SubmitTransaction(tx *transaction.Transaction) error
}
