# Stathera MVP: 3-Layer Monetary System

This is a Minimum Viable Product (MVP) implementation of the Stathera monetary system, focusing on a clean, three-layer architecture that prioritizes security and transaction speed.

## Architecture Overview

Stathera MVP is built around three distinct layers, each with a specific responsibility:

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                 Layer 3: Settlement                    │  │
│  │                                                        │  │
│  │  • Batch transaction finality                          │  │
│  │  • Cryptographic proofs (Merkle trees)                 │  │
│  │  • Ensures consistency between layers                  │  │
│  └───────────────────────────────────────────────────────┘  │
│                             ▲                               │
│                             │                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                Layer 2: Transaction                    │  │
│  │                                                        │  │
│  │  • High-speed transaction processing                   │  │
│  │  • Account management                                  │  │
│  │  • Signature validation                                │  │
│  └───────────────────────────────────────────────────────┘  │
│                             ▲                               │
│                             │                               │
│  ┌───────────────────────────────────────────────────────┐  │
│  │                 Layer 1: Ledger                        │  │
│  │                                                        │  │
│  │  • Canonical record of total supply                    │  │
│  │  • Immutable, cryptographically secure                 │  │
│  │  • Deterministic minting with controlled inflation     │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
└─────────────────────────────────────────────────────────────┘
                             ▲
                             │
┌─────────────────────────────────────────────────────────────┐
│                    Time Governance                           │
│                                                             │
│  • Self-contained, secure time proofs                       │
│  • Cryptographic verification                               │
│  • Cross-cutting concern for all layers                     │
└─────────────────────────────────────────────────────────────┘
```

### Layer 1: Canonical Ledger (Base Layer)

The foundational layer that:
- Stores the total monetary supply
- Provides immutable, cryptographically secure record-keeping
- Implements deterministic minting with simple annual issuance
- Maintains the source of truth for the entire system

### Layer 2: Transaction Engine (High-Speed Layer)

The middle layer that:
- Processes transactions with immediate validation
- Manages account balances and state
- Validates digital signatures
- Provides high-throughput, instant-confirmation design
- Serves as the primary interface for user operations

### Layer 3: Settlement Layer (Finality Layer)

The top layer that:
- Periodically finalizes batches of transactions into the canonical ledger
- Provides cryptographic proofs (Merkle trees) for security and verification
- Ensures eventual consistency between high-speed operations and the base ledger
- Enables auditability and long-term record keeping

### Time Governance Module

A cross-cutting module that:
- Provides an abstracted interface for time-related operations
- Implements secure, self-contained time proofs
- Supports the temporal aspects of all three layers

## Security Features

- **Cryptographic Verification**: Ed25519 signatures for transaction validation
- **Immutable Ledger**: Append-only design with hash-chain integrity
- **Time-Locked Proofs**: Secure time governance with cryptographic verification
- **Merkle Trees**: Efficient verification of transaction batches
- **Atomic Operations**: Thread-safe processing with proper locking

## Performance Optimizations

- **In-Memory Processing**: High-speed transaction layer operates in memory
- **Batch Settlement**: Efficient processing of transaction groups
- **Minimal Dependencies**: Focused implementation with few external libraries
- **Concurrent Design**: Thread-safe operations with proper synchronization

## Getting Started

### Prerequisites

- Go 1.24 or higher
- Git

### Building the Project

```bash
# Clone the repository
git clone https://github.com/cmatc13/stathera.git
cd stathera

# Build the MVP
go build -o stathera-mvp ./cmd/stathera-mvp
```

### Running the System

```bash
# Run with default settings
./stathera-mvp

# Run with custom settings
./stathera-mvp \
  --initial-supply 1000000 \
  --min-inflation 1.0 \
  --max-inflation 2.5 \
  --max-step-size 0.05 \
  --batch-size 500 \
  --settle-interval 1m
```

### Command-Line Options

- `--initial-supply`: Initial monetary supply (default: 20,000,000,000,000)
- `--min-inflation`: Minimum annual inflation rate in % (default: 1.5)
- `--max-inflation`: Maximum annual inflation rate in % (default: 3.0)
- `--max-step-size`: Maximum daily inflation adjustment in % (default: 0.1)
- `--batch-size`: Number of transactions per settlement batch (default: 1000)
- `--settle-interval`: Settlement interval (default: 5m)
- `--reserve-address`: Reserve account address (default: "RESERVE")
- `--fee-address`: Fee collection address (default: "FEES")

## Implementation Details

### Ledger Layer

The ledger layer maintains an immutable record of the total monetary supply and its changes over time. Each change is recorded as a ledger entry with:

- Timestamp
- Total supply
- Delta (change amount)
- Reason
- Cryptographic hash
- Previous entry hash (forming a chain)

This creates a verifiable chain of supply changes that can be audited at any time.

### Transaction Layer

The transaction layer handles the high-speed processing of financial transactions. It maintains:

- Account balances
- Transaction history
- Signature verification
- Nonce tracking (to prevent replay attacks)

Transactions go through a validation process that checks:
1. Signature validity
2. Sufficient funds
3. Nonce uniqueness
4. Basic transaction validity

### Settlement Layer

The settlement layer periodically takes batches of confirmed transactions and finalizes them by:

1. Creating a Merkle tree of transaction IDs
2. Generating a cryptographic time proof
3. Marking transactions as settled
4. Maintaining a chain of settlement batches

This creates a verifiable record of transaction finality that can be efficiently proven.

### Time Governance

The time governance module provides secure time proofs using HMAC-SHA256 signatures. Each time proof includes:

- Timestamp
- Nonce
- Cryptographic signature

This ensures that timestamps cannot be forged and provides a secure foundation for the entire system.

## Future Enhancements

While this MVP focuses on the core three-layer architecture, future enhancements could include:

1. **Distributed Processing**: Sharded transaction processing for horizontal scaling
2. **Advanced Cryptography**: Post-quantum cryptographic algorithms
3. **Smart Contracts**: Programmable transaction logic
4. **Enhanced Privacy**: Zero-knowledge proofs for transaction privacy
5. **Governance Mechanisms**: Decentralized control of system parameters

## License

This project is licensed under the MIT License - see the LICENSE file for details.
