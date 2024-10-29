package guardiansigner

import (
	"crypto/ecdsa"
	"fmt"
	"time"
)

type BenchmarkSigner struct {
	innerSigner GuardianSigner
}

func NewBenchmarkSigner(unsafeDevMode bool, signerKeyPath string) (*BenchmarkSigner, error) {
	innerSigner, err := NewGuardianSignerFromUri(signerKeyPath, unsafeDevMode)

	if err != nil {
		return nil, fmt.Errorf("failed to create benchmark signer: %w", err)
	}

	return &BenchmarkSigner{
		innerSigner: innerSigner,
	}, nil
}

func (b *BenchmarkSigner) Sign(hash []byte) ([]byte, error) {

	start := time.Now()

	sig, err := b.innerSigner.Sign(hash)

	duration := time.Since(start)
	fmt.Printf("Signing execution time: %v\n", duration)

	return sig, err
}

func (b *BenchmarkSigner) PublicKey() ecdsa.PublicKey {

	start := time.Now()

	pubKey := b.innerSigner.PublicKey()

	duration := time.Since(start)
	fmt.Printf("Public key retrieval time: %v\n", duration)

	return pubKey
}

func (b *BenchmarkSigner) Verify(sig []byte, hash []byte) (bool, error) {

	start := time.Now()

	valid, err := b.innerSigner.Verify(sig, hash)

	duration := time.Since(start)
	fmt.Printf("Signature verification time: %v\n", duration)

	return valid, err
}
