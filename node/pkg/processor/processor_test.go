package processor

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	tsscommon "github.com/xlabs/tss-common"
	"github.com/xlabs/tss-common/service/signer"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	tssmock "github.com/certusone/wormhole/node/pkg/tss/mock"
)

func TestHandleTssResponse(t *testing.T) {
	// Setup logger observer
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	observedLogger := zap.New(observedZapCore)

	// Helper to create processor
	createProcessor := func() *Processor {
		return &Processor{
			logger:          observedLogger,
			tssWaiters:      make(map[string]timedThresholdSignatureWaiter),
			updatedVAAs:     make(map[string]*updateVaaEntry),
			gossipVaaSendC:  make(chan []byte, 10),
			thresholdSigner: tssmock.NewMockSigner(),
			pythnetVaas:     make(map[string]PythNetVaaEntry),
		}
	}

	t.Run("NilResponse", func(t *testing.T) {
		p := createProcessor()
		p.handleTssResponse(nil)
		assert.Equal(t, 1, observedLogs.FilterMessage("received nil TSS signer response").Len())
		observedLogs.TakeAll()
	})

	t.Run("NilInnerResponse", func(t *testing.T) {
		p := createProcessor()
		p.handleTssResponse(&signer.SignResponse{Response: nil})
		assert.Equal(t, 1, observedLogs.FilterMessage("received nil TSS signer response").Len())
		observedLogs.TakeAll()
	})

	t.Run("NilSignature", func(t *testing.T) {
		p := createProcessor()
		resp := &signer.SignResponse{
			Response: &signer.SignResponse_Signature{
				Signature: nil,
			},
		}
		p.handleTssResponse(resp)
		assert.Equal(t, 1, observedLogs.FilterMessage("received nil TSS signer signature").Len())
		observedLogs.TakeAll()
	})

	t.Run("NilStatus", func(t *testing.T) {
		p := createProcessor()
		resp := &signer.SignResponse{
			Response: &signer.SignResponse_Status{
				Status: nil,
			},
		}
		p.handleTssResponse(resp)
		assert.Equal(t, 1, observedLogs.FilterMessage("received nil TSS signer status").Len())
		observedLogs.TakeAll()
	})

	t.Run("ProcessStatus", func(t *testing.T) {
		p := createProcessor()
		digest := []byte("digest")
		hash := hex.EncodeToString(digest)

		// Add to waiters
		p.tssWaiters[hash] = timedThresholdSignatureWaiter{
			startTime: time.Now(),
			vaa:       &VAA{},
		}

		resp := &signer.SignResponse{
			Response: &signer.SignResponse_Status{
				Status: &signer.SignStatus{
					Digest:   digest,
					Code:     1, // Error code
					Message:  "failed",
					Protocol: "frost",
				},
			},
		}

		p.handleTssResponse(resp)

		// Should be removed from waiters
		_, exists := p.tssWaiters[hash]
		assert.False(t, exists)
		assert.Equal(t, 1, observedLogs.FilterMessage("TSS signing request stopped").Len())
		observedLogs.TakeAll()
	})

	t.Run("ProcessStatus_NilDigest", func(t *testing.T) {
		p := createProcessor()
		resp := &signer.SignResponse{
			Response: &signer.SignResponse_Status{
				Status: &signer.SignStatus{
					Digest: nil,
				},
			},
		}
		p.handleTssResponse(resp)
		assert.Equal(t, 1, observedLogs.FilterMessage("received nil TSS signer status").Len())
		observedLogs.TakeAll()
	})

	t.Run("ProcessStatus_ZeroCode", func(t *testing.T) {
		p := createProcessor()
		resp := &signer.SignResponse{
			Response: &signer.SignResponse_Status{
				Status: &signer.SignStatus{
					Digest: []byte("digest"),
					Code:   0,
				},
			},
		}
		p.handleTssResponse(resp)
		assert.Equal(t, 1, observedLogs.FilterMessage("received nil TSS signer status").Len())
		observedLogs.TakeAll()
	})

	t.Run("ProcessSignature_NilS", func(t *testing.T) {
		p := createProcessor()
		resp := &signer.SignResponse{
			Response: &signer.SignResponse_Signature{
				Signature: &tsscommon.SignatureData{
					M: []byte("digest"),
					S: nil,
				},
			},
		}
		p.handleTssResponse(resp)
		assert.Equal(t, 1, observedLogs.FilterMessage("received TSS signature with nil signature").Len())
		observedLogs.TakeAll()
	})

	t.Run("ProcessSignature_NilM", func(t *testing.T) {
		p := createProcessor()
		resp := &signer.SignResponse{
			Response: &signer.SignResponse_Signature{
				Signature: &tsscommon.SignatureData{
					M: nil,
					S: []byte("sig"),
				},
			},
		}
		p.handleTssResponse(resp)
		assert.Equal(t, 1, observedLogs.FilterMessage("received TSS signature with nil message").Len())
		observedLogs.TakeAll()
	})

	t.Run("ProcessSignature_UnknownVAA", func(t *testing.T) {
		p := createProcessor()
		resp := &signer.SignResponse{
			Response: &signer.SignResponse_Signature{
				Signature: &tsscommon.SignatureData{
					M: []byte("unknown"),
					S: []byte("sig"),
				},
			},
		}
		p.handleTssResponse(resp)
		assert.Equal(t, 1, observedLogs.FilterMessage("received TSS signature for unknown VAA").Len())
		observedLogs.TakeAll()
	})

	t.Run("ProcessSignature_InvalidSig", func(t *testing.T) {
		p := createProcessor()
		digest := []byte("digest_for_sig")
		hash := hex.EncodeToString(digest)

		v := &VAA{
			VAA: vaa.VAA{
				EmitterChain: vaa.ChainIDSolana,
			},
		}

		p.tssWaiters[hash] = timedThresholdSignatureWaiter{
			startTime: time.Now(),
			vaa:       v,
		}

		resp := &signer.SignResponse{
			Response: &signer.SignResponse_Signature{
				Signature: &tsscommon.SignatureData{
					M: digest,
					S: []byte("invalid_frost_sig"),
				},
			},
		}
		p.handleTssResponse(resp)
		assert.Equal(t, 1, observedLogs.FilterMessage("failed to translate TSS signature").Len())
		observedLogs.TakeAll()
	})

	t.Run("ProcessSignature_Success", func(t *testing.T) {
		p := createProcessor()
		digest := []byte("digest_for_sig")
		hash := hex.EncodeToString(digest)

		v := &VAA{
			VAA: vaa.VAA{
				EmitterChain: vaa.ChainIDSolana,
			},
		}

		p.tssWaiters[hash] = timedThresholdSignatureWaiter{
			startTime: time.Now(),
			vaa:       v,
		}
		sigResp := tssmock.NewMockSigner().Sign(digest, tsscommon.ProtocolFROSTSign) // Pre-initialize mock signer

		// Construct a dummy frost signature

		// Mock the witness call on the signer

		p.handleTssResponse(sigResp)

		// Check logs for success message from HandleQuorum
		assert.Equal(t, 1, observedLogs.FilterMessage("signed VAA with quorum").Len())

		// Verify VAA was stored
		p.updateVAALock.Lock()

		assert.NotEmpty(t, p.updatedVAAs)
		p.updateVAALock.Unlock()

		observedLogs.TakeAll()
	})
}
