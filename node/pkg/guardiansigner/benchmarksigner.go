package guardiansigner

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type BenchmarkSigner struct {
	innerSigner GuardianSigner
}

var (
	guardianSignerSigningLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "wormhole_guardian_signer_signing_latency_us",
			Help:    "Latency histogram for Guardian signing requests",
			Buckets: []float64{10.0, 20.0, 50.0, 100.0, 1000.0, 5000.0, 10000.0, 100_000.0, 1_000_000.0, 10_000_000.0, 100_000_000.0, 1_000_000_000.0},
		})

	guardianSignerVerifyLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "wormhole_guardian_signer_sig_verify_latency_us",
			Help:    "Latency histogram for Guardian signature verification requests",
			Buckets: []float64{10.0, 20.0, 50.0, 100.0, 1000.0, 5000.0, 10000.0, 100_000.0, 1_000_000.0, 10_000_000.0, 100_000_000.0, 1_000_000_000.0},
		})
)

// The Benchmark signer simply wraps any other available signer, but records the latency of signing and signature
// verification operations into histograms. To use the Benchmark signer, use the `benchmark://` URI scheme, followed
// by the signer URI that needs to be benchmarked. For example:
//
//	benchmark://amazonkms://<arn>
func NewBenchmarkSigner(ctx context.Context, unsafeDevMode bool, signerKeyPath string) (*BenchmarkSigner, error) {
	innerSigner, err := NewGuardianSignerFromUri(ctx, signerKeyPath, unsafeDevMode)

	if err != nil {
		return nil, fmt.Errorf("failed to create benchmark signer: %w", err)
	}

	return &BenchmarkSigner{
		innerSigner: innerSigner,
	}, nil
}

func (b *BenchmarkSigner) Sign(ctx context.Context, hash []byte) ([]byte, error) {

	start := time.Now()
	sig, err := b.innerSigner.Sign(ctx, hash)
	duration := time.Since(start)

	// Add Observation to histogram
	guardianSignerSigningLatency.Observe(float64(duration.Microseconds()))

	return sig, err
}

func (b *BenchmarkSigner) PublicKey(ctx context.Context) ecdsa.PublicKey {
	pubKey := b.innerSigner.PublicKey(ctx)
	return pubKey
}

func (b *BenchmarkSigner) Verify(ctx context.Context, sig []byte, hash []byte) (bool, error) {

	start := time.Now()
	valid, err := b.innerSigner.Verify(ctx, sig, hash)
	duration := time.Since(start)

	// Add observation to histogram
	guardianSignerVerifyLatency.Observe(float64(duration.Microseconds()))

	return valid, err
}
