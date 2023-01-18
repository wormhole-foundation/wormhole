package accountant

import (
	// "encoding/hex"
	"encoding/json"
	"testing"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	// wasmdtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseObservationResponseDataKey(t *testing.T) {
	dataJson := []byte("{\"emitter_chain\":2,\"emitter_address\":\"AAAAAAAAAAAAAAAAApD7FnIIr0VbsTd4AWO3t6mhDBY=\",\"sequence\":1673978163}")

	var key ObservationKey
	err := json.Unmarshal(dataJson, &key)
	require.NoError(t, err)

	expectedEmitterAddress, err := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	expectedResult := ObservationKey{
		EmitterChain:   uint16(vaa.ChainIDEthereum),
		EmitterAddress: expectedEmitterAddress.Bytes(),
		Sequence:       1673978163,
	}
	assert.Equal(t, expectedResult, key)
}

func TestParseObservationResponseData(t *testing.T) {

	/*
		executeContractJson := []byte("\n\ufffd\u0002[{\"key\":{\"emitter_chain\":2,\"emitter_address\":\"AAAAAAAAAAAAAAAAApD7FnIIr0VbsTd4AWO3t6mhDBY=\",\"sequence\":1674061268},\"status\":{\"type\":\"committed\"}},{\"key\":{\"emitter_chain\":2,\"emitter_address\":\"AAAAAAAAAAAAAAAAApD7FnIIr0VbsTd4AWO3t6mhDBY=\",\"sequence\":1674061267},\"status\":{\"type\":\"error\",\"data\":\"digest mismatch for processed message\"}}]")
		// executeContractHex := []byte("0AFA020A242F636F736D7761736D2E7761736D2E76312E4D736745786563757465436F6E747261637412D1020ACE025B7B226B6579223A7B22656D69747465725F636861696E223A322C22656D69747465725F61646472657373223A224141414141414141414141414141414141704437466E4949723056627354643441574F3374366D684442593D222C2273657175656E6365223A313637343036313236387D2C22737461747573223A7B2274797065223A22636F6D6D6974746564227D7D2C7B226B6579223A7B22656D69747465725F636861696E223A322C22656D69747465725F61646472657373223A224141414141414141414141414141414141704437466E4949723056627354643441574F3374366D684442593D222C2273657175656E6365223A313637343036313236377D2C22737461747573223A7B2274797065223A226572726F72222C2264617461223A22646967657374206D69736D6174636820666F722070726F636573736564206D657373616765227D7D5D")
		var ec wasmdtypes.MsgExecuteContractResponse
		err := json.Unmarshal(executeContractJson, &ec)
		require.NoError(t, err)
	*/

	responsesJson := []byte("[{\"key\":{\"emitter_chain\":2,\"emitter_address\":\"AAAAAAAAAAAAAAAAApD7FnIIr0VbsTd4AWO3t6mhDBY=\",\"sequence\":1674061268},\"status\":{\"type\":\"committed\"}},{\"key\":{\"emitter_chain\":2,\"emitter_address\":\"AAAAAAAAAAAAAAAAApD7FnIIr0VbsTd4AWO3t6mhDBY=\",\"sequence\":1674061267},\"status\":{\"type\":\"error\",\"data\":\"digest mismatch for processed message\"}}]")
	var responses ObservationResponses
	err := json.Unmarshal(responsesJson, &responses)
	require.NoError(t, err)
	require.Equal(t, 2, len(responses))

	expectedEmitterAddress, err := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	expectedResult0 := ObservationResponse{
		Key: ObservationKey{
			EmitterChain:   uint16(vaa.ChainIDEthereum),
			EmitterAddress: expectedEmitterAddress.Bytes(),
			Sequence:       1674061268,
		},
		Status: ObservationResponseStatus{
			Type: "committed",
		},
	}

	expectedResult1 := ObservationResponse{
		Key: ObservationKey{
			EmitterChain:   uint16(vaa.ChainIDEthereum),
			EmitterAddress: expectedEmitterAddress.Bytes(),
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
