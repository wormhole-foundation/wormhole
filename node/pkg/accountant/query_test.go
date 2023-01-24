package accountant

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	cosmossdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMissingObservationsResponse(t *testing.T) {
	//TODO: Write this test once we get a sample response.
}

func TestParseBatchTransferStatusResponse(t *testing.T) {
	responsesJson := []byte("{\"details\":[{\"key\":{\"emitter_chain\":2,\"emitter_address\":\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\",\"sequence\":1674568234},\"status\":{\"committed\":{\"data\":{\"amount\":\"1000000000000000000\",\"token_chain\":2,\"token_address\":\"0000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a\",\"recipient_chain\":4},\"digest\":\"1nbbff/7/ai9GJUs4h2JymFuO4+XcasC6t05glXc99M=\"}}},{\"key\":{\"emitter_chain\":2,\"emitter_address\":\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\",\"sequence\":1674484597},\"status\":null}]}")
	var response BatchTransferStatusResponse
	err := json.Unmarshal(responsesJson, &response)
	require.NoError(t, err)
	require.Equal(t, 2, len(response.Details))

	expectedEmitterAddress, err := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	expectedTokenAddress, err := vaa.StringToAddress("0000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a")
	require.NoError(t, err)

	expectedAmount0 := cosmossdk.NewInt(1000000000000000000)

	expectedDigest0, err := hex.DecodeString("d676db7dfffbfda8bd18952ce21d89ca616e3b8f9771ab02eadd398255dcf7d3")
	require.NoError(t, err)

	expectedResult0 := TransferDetails{
		Key: TransferKey{
			EmitterChain:   uint16(vaa.ChainIDEthereum),
			EmitterAddress: expectedEmitterAddress,
			Sequence:       1674568234,
		},
		Status: TransferStatus{
			Committed: &TransferStatusCommitted{
				Data: TransferData{
					Amount:         &expectedAmount0,
					TokenChain:     uint16(vaa.ChainIDEthereum),
					TokenAddress:   expectedTokenAddress,
					RecipientChain: uint16(vaa.ChainIDBSC),
				},
				Digest: expectedDigest0,
			},
		},
	}

	expectedResult1 := TransferDetails{
		Key: TransferKey{
			EmitterChain:   uint16(vaa.ChainIDEthereum),
			EmitterAddress: expectedEmitterAddress,
			Sequence:       1674484597,
		},
		Status: TransferStatus{},
	}

	require.NotNil(t, response.Details[0].Status.Committed)
	require.Nil(t, response.Details[0].Status.Pending)
	assert.True(t, reflect.DeepEqual(expectedResult0, response.Details[0]))

	// Use DeepEqual() because the response contains pointers.
	assert.True(t, reflect.DeepEqual(expectedResult0, response.Details[0]))
	assert.True(t, reflect.DeepEqual(expectedResult1, response.Details[1]))
}
