// internal/wallet/wallet.go
package wallet

import (
    "crypto/rand"
    "crypto/sha256"
    "encoding/hex"
    "errors"
    "fmt"
    "time"

    "github.com/btcsuite/btcd/btcec/v2"
    "github.com/btcsuite/btcd/btcec/v2/ecdsa"
    "github.com/btcsuite/btcutil/base58"
)

// Wallet represents a user's digital wallet
type Wallet struct {
    PrivateKey *btcec.PrivateKey
    PublicKey  []byte
    Address    string
    CreatedAt  time.Time
}

// NewWallet creates a new wallet with a unique key pair
func NewWallet() (*Wallet, error) {
    privateKey, err := btcec.NewPrivateKey()
    if err != nil {
        return nil, fmt.Errorf("failed to generate private key: %w", err)
    }

    pubKey := privateKey.PubKey().SerializeCompressed()
    
    // Create a more robust address
    sha := sha256.New()
    sha.Write(pubKey)
    hash := sha.Sum(nil)
    address := base58.Encode(hash[:20]) // Use first 20 bytes for address (Bitcoin-like)
    
    return &Wallet{
        PrivateKey: privateKey,
        PublicKey:  pubKey,
        Address:    address,
        CreatedAt:  time.Now(),
    }, nil
}

// ExportPrivateKey exports the private key as a hex string
func (w *Wallet) ExportPrivateKey() string {
    return hex.EncodeToString(w.PrivateKey.Serialize())
}

// ImportWallet imports a wallet from a private key hex string
func ImportWallet(privateKeyHex string) (*Wallet, error) {
    privateKeyBytes, err := hex.DecodeString(privateKeyHex)
    if err != nil {
        return nil, fmt.Errorf("invalid private key format: %w", err)
    }
    
    privateKey, _ := btcec.PrivKeyFromBytes(privateKeyBytes)
    if privateKey == nil {
        return nil, errors.New("invalid private key")
    }

    pubKey := privateKey.PubKey().SerializeCompressed()
    
    // Create address
    sha := sha256.New()
    sha.Write(pubKey)
    hash := sha.Sum(nil)
    address := base58.Encode(hash[:20]) 
    
    return &Wallet{
        PrivateKey: privateKey,
        PublicKey:  pubKey,
        Address:    address,
        CreatedAt:  time.Now(),
    }, nil
}

// SignMessage signs a message with the wallet's private key
func (w *Wallet) SignMessage(message []byte) ([]byte, error) {
    signature := ecdsa.Sign(w.PrivateKey, message)
    return signature.Serialize(), nil
}

// VerifySignature verifies a signature against the wallet's public key
func VerifySignature(pubKey, message, signature []byte) (bool, error) {
    parsedPubKey, err := btcec.ParsePubKey(pubKey)
    if err != nil {
        return false, fmt.Errorf("failed to parse public key: %w", err)
    }
    
    parsedSig, err := ecdsa.ParseSignature(signature)
    if err != nil {
        return false, fmt.Errorf("failed to parse signature: %w", err)
    }
    
    return parsedSig.Verify(message, parsedPubKey), nil
}

// GenerateNonce generates a secure random nonce
func GenerateNonce() (string, error) {
    nonce := make([]byte, 16)
    _, err := rand.Read(nonce)
    if err != nil {
        return "", err
    }
    return hex.EncodeToString(nonce), nil
}