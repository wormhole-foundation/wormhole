package accountant

import (
	// "encoding/hex"
	"encoding/json"
	"testing"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseObservationResponseDataKey(t *testing.T) {
	dataJson := []byte("{\"emitter_chain\":2,\"emitter_address\":\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\",\"sequence\":1673978163}")

	var key TransferKey
	err := json.Unmarshal(dataJson, &key)
	require.NoError(t, err)

	expectedEmitterAddress, err := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	expectedResult := TransferKey{
		EmitterChain:   uint16(vaa.ChainIDEthereum),
		EmitterAddress: expectedEmitterAddress,
		Sequence:       1673978163,
	}
	assert.Equal(t, expectedResult, key)
}

func TestParseObservationResponseData(t *testing.T) {
	responsesJson := []byte("[{\"key\":{\"emitter_chain\":2,\"emitter_address\":\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\",\"sequence\":1674061268},\"status\":{\"type\":\"committed\"}},{\"key\":{\"emitter_chain\":2,\"emitter_address\":\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\",\"sequence\":1674061267},\"status\":{\"type\":\"error\",\"data\":\"digest mismatch for processed message\"}}]")
	var responses ObservationResponses
	err := json.Unmarshal(responsesJson, &responses)
	require.NoError(t, err)
	require.Equal(t, 2, len(responses))

	expectedEmitterAddress, err := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	expectedResult0 := ObservationResponse{
		Key: TransferKey{
			EmitterChain:   uint16(vaa.ChainIDEthereum),
			EmitterAddress: expectedEmitterAddress,
			Sequence:       1674061268,
		},
		Status: ObservationResponseStatus{
			Type: "committed",
		},
	}

	expectedResult1 := ObservationResponse{
		Key: TransferKey{
			EmitterChain:   uint16(vaa.ChainIDEthereum),
			EmitterAddress: expectedEmitterAddress,
			Sequence:       1674061267,
		},
		Status: ObservationResponseStatus{
			Type: "error",
			Data: "digest mismatch for processed message",
		},
	}

	assert.Equal(t, expectedResult0, responses[0])
	assert.Equal(t, expectedResult1, responses[1])
}
