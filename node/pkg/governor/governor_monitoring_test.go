package governor

import (
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/devnet"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestIsVAAEnqueuedNilMessageID(t *testing.T) {
	logger, _ := zap.NewProduction()
	gk := devnet.InsecureDeterministicEcdsaKeyByIndex(ethCrypto.S256(), uint64(0))
	gst := common.NewGuardianSetState(nil)
	gs := &common.GuardianSet{Keys: []ethCommon.Address{ethCommon.HexToAddress("0xbeFA429d57cD18b7F8A4d91A2da9AB4AF05d0FBe")}}
	gst.Set(gs)
	gov := NewChainGovernor(logger, nil, ethCrypto.PubkeyToAddress(gk.PublicKey), gst, common.GoTest)
	enqueued, err := gov.IsVAAEnqueued(nil)
	require.EqualError(t, err, "no message ID specified")
	assert.Equal(t, false, enqueued)
}
