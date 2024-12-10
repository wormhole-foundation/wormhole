package wormchain_ibc_receiver

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func SubmitIbcReceiverUpdateChannelChainMsg(t *testing.T,
	allowlistChainID vaa.ChainID,
	allowlistChannel string,
	guardians *guardians.ValSet) string {

	paddedChannel, _ := vaa.LeftPadIbcChannelId(allowlistChannel)

	bodyIbcReceiverUpdateChannelChain := vaa.BodyIbcUpdateChannelChain{
		TargetChainId: vaa.ChainIDWormchain,
		ChannelId:     paddedChannel,
		ChainId:       allowlistChainID,
	}

	payload, err := bodyIbcReceiverUpdateChannelChain.Serialize(vaa.IbcReceiverModuleStr)
	require.NoError(t, err)

	v := helpers.GenerateGovernanceVaa(0, guardians, payload)
	vBz, err := v.Marshal()
	require.NoError(t, err)
	vHex := base64.StdEncoding.EncodeToString(vBz)

	var vaas [1]Binary
	vaas[0] = Binary(vHex)

	submitVAAMsg := ExecuteMsg{
		SubmitUpdateChannelChain: &ExecuteMsg_SubmitUpdateChannelChain{
			Vaas: vaas[:],
		},
	}

	submitVAAMsgBz, err := json.Marshal(submitVAAMsg)
	require.NoError(t, err)

	return string(submitVAAMsgBz)
}

type ReceiverAck struct {
	Ok    *struct{} `json:"ok,omitempty"`
	Error string    `json:"error,omitempty"`
}

func (r ReceiverAck) IsOk() bool {
	return len(r.Error) == 0
}

func (r ReceiverAck) IsError() bool {
	return len(r.Error) > 0
}
