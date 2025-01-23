package helpers

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
)

var latestSequence = 1

func signVaa(vaaToSign vaa.VAA, signers *guardians.ValSet) vaa.VAA {
	for i, key := range signers.Vals {
		vaaToSign.AddSignature(key.Priv, uint8(i))
	}
	return vaaToSign
}

func GenerateVaa(index uint32, signers *guardians.ValSet, emitterChain vaa.ChainID, emitterAddr vaa.Address, payload []byte) vaa.VAA {
	v := vaa.VAA{
		Version:          uint8(1),
		GuardianSetIndex: index,
		Signatures:       nil,
		Timestamp:        time.Unix(0, 0),
		Nonce:            uint32(1),
		Sequence:         uint64(latestSequence),
		ConsistencyLevel: uint8(32),
		EmitterChain:     emitterChain,
		EmitterAddress:   emitterAddr,
		Payload:          payload,
	}
	latestSequence = latestSequence + 1
	return signVaa(v, signers)
}

func GenerateGovernanceVaa(index uint32,
	signers *guardians.ValSet,
	payload []byte) vaa.VAA {

	v := vaa.CreateGovernanceVAA(time.Unix(0, 0),
		uint32(1), uint64(latestSequence), index, payload)

	latestSequence = latestSequence + 1
	return signVaa(*v, signers)
}

func GenerateEmptyVAA(
	t *testing.T,
	guardians *guardians.ValSet,
	moduleStr string,
	action vaa.GovernanceAction,
	chainID vaa.ChainID,
) string {

	payloadBz, err := vaa.EmptyPayloadVaa(moduleStr, action, chainID)
	require.NoError(t, err)
	v := generateVaa(0, guardians, vaa.GovernanceChain, vaa.GovernanceEmitter, payloadBz)

	v = signVaa(v, guardians)
	vBz, err := v.Marshal()
	require.NoError(t, err)
	vHex := hex.EncodeToString(vBz)

	return vHex
}
