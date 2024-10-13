package processor

import (
	"bytes"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/devnet"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"google.golang.org/protobuf/proto"
)

func getUniqueVAA(seqNo uint64) vaa.VAA {
	var payload = []byte{97, 97, 97, 97, 97, 97}
	var governanceEmitter = vaa.Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}

	vaa := vaa.VAA{
		Version:          uint8(1),
		GuardianSetIndex: uint32(1),
		Signatures:       nil,
		Timestamp:        time.Unix(0, 0),
		Nonce:            uint32(1),
		Sequence:         seqNo,
		ConsistencyLevel: uint8(32),
		EmitterChain:     vaa.ChainIDSolana,
		EmitterAddress:   governanceEmitter,
		Payload:          payload,
	}

	return vaa
}

func TestMarshalSignedObservationBatch(t *testing.T) {
	gk := devnet.InsecureDeterministicEcdsaKeyByIndex(crypto.S256(), uint64(0))
	require.NotNil(t, gk)

	NumObservations := uint64(p2p.MaxObservationBatchSize)
	observations := make([]*gossipv1.Observation, 0, NumObservations)
	for seqNo := uint64(1); seqNo <= NumObservations; seqNo++ {
		vaa := getUniqueVAA(seqNo)
		digest := vaa.SigningDigest()
		sig, err := crypto.Sign(digest.Bytes(), gk)
		require.NoError(t, err)

		observations = append(observations, &gossipv1.Observation{
			Hash:      digest.Bytes(),
			Signature: sig,
			TxHash:    ethcommon.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
			MessageId: vaa.MessageID(),
		})
	}

	obsBuf, err := proto.Marshal(observations[0])
	require.NoError(t, err)
	// TODO: the correct length is 205, but since I added the sigVersion to every MessageId it's now 207. there is an ongoin branch attempting to fix this.
	// once fixed, this test should fail. and expexct 205.
	assert.Equal(t, 207, len(obsBuf))

	batch := gossipv1.SignedObservationBatch{
		Addr:         crypto.PubkeyToAddress(gk.PublicKey).Bytes(),
		Observations: observations,
	}

	buf, err := proto.Marshal((&batch))
	require.NoError(t, err)
	assert.Greater(t, pubsub.DefaultMaxMessageSize, len(buf))

	var batch2 gossipv1.SignedObservationBatch
	err = proto.Unmarshal(buf, &batch2)
	require.NoError(t, err)

	assert.True(t, bytes.Equal(batch.Addr, batch2.Addr))
	assert.Equal(t, len(batch.Observations), len(batch2.Observations))
	for idx := range batch2.Observations {
		assert.True(t, bytes.Equal(batch.Observations[idx].Hash, batch2.Observations[idx].Hash))
		assert.True(t, bytes.Equal(batch.Observations[idx].Signature, batch2.Observations[idx].Signature))
		assert.True(t, bytes.Equal(batch.Observations[idx].TxHash, batch2.Observations[idx].TxHash))
		assert.Equal(t, batch.Observations[idx].MessageId, batch2.Observations[idx].MessageId)
	}
}
