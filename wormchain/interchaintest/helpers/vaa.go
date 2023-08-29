package helpers

import (
	"time"

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

func generateVaa(index uint32, signers *guardians.ValSet, emitterChain vaa.ChainID, emitterAddr vaa.Address, payload []byte) vaa.VAA {
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
