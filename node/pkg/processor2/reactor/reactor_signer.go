package reactor

import (
	"context"
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Signer enables cryptographic operations of a guardian in consensus
type Signer interface {
	// Sign signs a digest using the guardian key
	Sign(ctx context.Context, digest []byte) (signature []byte, err error)
	// Address returns the guardian key
	Address(ctx context.Context) (common.Address, error)
}

// EcdsaKeySigner implements Signer using an in-memory ecdsa key
type EcdsaKeySigner struct {
	gk *ecdsa.PrivateKey
}

// NewEcdsaKeySigner creates a new EcdsaKeySigner
func NewEcdsaKeySigner(key *ecdsa.PrivateKey) *EcdsaKeySigner {
	return &EcdsaKeySigner{gk: key}
}

func (e *EcdsaKeySigner) Sign(_ context.Context, digest []byte) (signature []byte, err error) {
	return crypto.Sign(digest, e.gk)
}

func (e *EcdsaKeySigner) Address(_ context.Context) (common.Address, error) {
	return crypto.PubkeyToAddress(e.gk.PublicKey), nil
}
