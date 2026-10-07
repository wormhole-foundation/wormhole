package processor

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/db"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// TestHandleCleanup_RetryRequestsReobservationExceptForGovernance checks the retry of an unsubmitted observation
// of our own: it always rebroadcasts our observation, but requests a re-observation only for messages that have a
// source transaction. A governance message is injected with a zero TxID, so a request for it can't be served.
func TestHandleCleanup_RetryRequestsReobservationExceptForGovernance(t *testing.T) {
	tokenBridge, err := vaa.StringToAddress("0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585")
	require.NoError(t, err)

	tests := []struct {
		name        string
		chain       vaa.ChainID
		emitter     vaa.Address
		txHash      []byte
		wantRequest bool
	}{
		{
			name:        "governance message",
			chain:       vaa.GovernanceChain,
			emitter:     vaa.GovernanceEmitter,
			txHash:      ethcommon.Hash{}.Bytes(),
			wantRequest: false,
		},
		{
			name:        "ordinary Solana message",
			chain:       vaa.ChainIDSolana,
			emitter:     tokenBridge,
			txHash:      ethcommon.HexToHash("0x01").Bytes(),
			wantRequest: true,
		},
		{
			name:        "governance emitter address on another chain",
			chain:       vaa.ChainIDEthereum,
			emitter:     vaa.GovernanceEmitter,
			txHash:      ethcommon.HexToHash("0x02").Bytes(),
			wantRequest: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			guardianDB := db.OpenDb(zap.NewNop(), nil)
			t.Cleanup(func() { _ = guardianDB.Close() })

			obsvReqSendC := make(chan *gossipv1.ObservationRequest, 1)
			batchObsvPubC := make(chan *gossipv1.Observation, 1)
			p := &Processor{
				logger:        zap.NewNop(),
				db:            guardianDB,
				obsvReqSendC:  obsvReqSendC,
				batchObsvPubC: batchObsvPubC,
				state:         &aggregationState{signatures: observationMap{}},
				delegateState: &delegateAggregationState{observations: delegateObservationMap{}},
				updateVAALock: sync.Mutex{},
				updatedVAAs:   make(map[string]*updateVaaEntry),
			}

			v := &VAA{VAA: vaa.VAA{
				Version:          vaa.SupportedVAAVersion,
				Timestamp:        time.Unix(1_700_000_000, 0),
				Nonce:            1,
				Sequence:         1936730226007234648,
				ConsistencyLevel: 32,
				EmitterChain:     tc.chain,
				EmitterAddress:   tc.emitter,
				Payload:          []byte{1, 2, 3},
			}}
			ourObs := &gossipv1.Observation{Hash: v.SigningDigest().Bytes(), TxHash: tc.txHash, MessageId: v.MessageID()}
			s := &state{
				firstObserved:  time.Now().Add(-2 * FirstRetryMinWait),
				ourObservation: v,
				signatures:     map[ethcommon.Address][]byte{},
				settled:        true,
				ourObs:         ourObs,
				txHash:         tc.txHash,
			}
			p.state.signatures[v.SigningDigest().Hex()] = s

			p.handleCleanup(context.Background())

			// The retry happened either way: our observation is rebroadcast and the backoff advances.
			require.Len(t, batchObsvPubC, 1, "our observation must be rebroadcast")
			assert.Equal(t, ourObs, <-batchObsvPubC)
			assert.Equal(t, uint(1), s.retryCtr)
			assert.True(t, s.nextRetry.After(time.Now()))

			if !tc.wantRequest {
				assert.Empty(t, obsvReqSendC, "no re-observation request for a governance message")
				return
			}
			require.Len(t, obsvReqSendC, 1)
			req := <-obsvReqSendC
			assert.Equal(t, uint32(tc.chain), req.ChainId)
			assert.Equal(t, tc.txHash, req.TxHash)
		})
	}
}
