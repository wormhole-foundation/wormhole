package guardiansigner

/*
	The Benchmark signer is a type of signer that wraps other signers,
	recording the latency of signing and signature verification into
	histograms. As additional signers are implemented, relying on 3rd
	party services, benchmarking signers is useful to ensure observation
	signing happens at an acceptable rate.
*/

import (
	"context"
	"crypto/ecdsa"
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

func BenchmarkWrappedSigner(innerSigner GuardianSigner) *BenchmarkSigner {
	if innerSigner == nil {
		return nil
	}

	return &BenchmarkSigner{
		innerSigner: innerSigner,
	}
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
