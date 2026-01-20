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

// The BenchmarkSigner is a signer that wraps other signers, recording the latency of
// signing and signature verification through prometheus histograms.
type BenchmarkSigner struct {
	innerSigner GuardianSigner

	// Per-instance metrics (labeled from the vectors)
	signingLatency    prometheus.Observer
	signingErrorCount prometheus.Counter
	verifyLatency     prometheus.Observer
	verifyErrorCount  prometheus.Counter
}

// Package-level metric vectors registered once
var (
	guardianSignerSigningLatencyVec = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wormhole_guardian_signer_signing_latency_us",
			Help:    "Latency histogram for Guardian signing requests",
			Buckets: []float64{10.0, 20.0, 50.0, 100.0, 1000.0, 5000.0, 10000.0, 100_000.0, 1_000_000.0, 10_000_000.0, 100_000_000.0, 1_000_000_000.0},
		},
		[]string{"signer_type", "purpose"},
	)

	guardianSignerSigningErrorCountVec = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_guardian_signer_signing_error_count",
			Help: "Total number of errors that occurred during Guardian signing requests",
		},
		[]string{"signer_type", "purpose"},
	)

	guardianSignerVerifyLatencyVec = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "wormhole_guardian_signer_sig_verify_latency_us",
			Help:    "Latency histogram for Guardian signature verification requests",
			Buckets: []float64{10.0, 20.0, 50.0, 100.0, 1000.0, 5000.0, 10000.0, 100_000.0, 1_000_000.0, 10_000_000.0, 100_000_000.0, 1_000_000_000.0},
		},
		[]string{"signer_type", "purpose"},
	)

	guardianSignerVerifyErrorCountVec = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_guardian_signer_verify_error_count",
			Help: "Total number of errors that occurred during Guardian signature verification requests",
		},
		[]string{"signer_type", "purpose"},
	)
)

// BenchmarkWrappedSigner wraps a signer with benchmarking metrics.
// The purpose parameter distinguishes different uses (e.g., "guardian", "manager").
func BenchmarkWrappedSigner(innerSigner GuardianSigner) *BenchmarkSigner {
	return BenchmarkWrappedSignerWithPurpose(innerSigner, "guardian")
}

// BenchmarkWrappedSignerWithPurpose wraps a signer with benchmarking metrics and a custom purpose label.
func BenchmarkWrappedSignerWithPurpose(innerSigner GuardianSigner, purpose string) *BenchmarkSigner {
	if innerSigner == nil {
		return nil
	}

	signerType := innerSigner.TypeAsString()

	return &BenchmarkSigner{
		innerSigner:       innerSigner,
		signingLatency:    guardianSignerSigningLatencyVec.WithLabelValues(signerType, purpose),
		signingErrorCount: guardianSignerSigningErrorCountVec.WithLabelValues(signerType, purpose),
		verifyLatency:     guardianSignerVerifyLatencyVec.WithLabelValues(signerType, purpose),
		verifyErrorCount:  guardianSignerVerifyErrorCountVec.WithLabelValues(signerType, purpose),
	}
}

func (b *BenchmarkSigner) Sign(ctx context.Context, hash []byte) ([]byte, error) {
	start := time.Now()
	sig, err := b.innerSigner.Sign(ctx, hash)
	duration := time.Since(start)

	// If an error occurred, increment the error counter
	if err != nil {
		b.signingErrorCount.Inc()
	} else {
		// Add Observation to histogram only if no errors occurred
		b.signingLatency.Observe(float64(duration.Microseconds()))
	}

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

	// If an error occurred, increment the error counter
	if err != nil {
		b.verifyErrorCount.Inc()
	} else {
		// Add observation to histogram only if no errors occurred
		b.verifyLatency.Observe(float64(duration.Microseconds()))
	}

	return valid, err
}

// Return the type of signer as "benchmark".
func (b *BenchmarkSigner) TypeAsString() string {
	return "benchmark"
}
