package keeper

import (
	"bytes"
	"encoding/binary"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormhole-chain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func ParseVAA(data []byte) (*vaa.VAA, error) {
	v, err := vaa.Unmarshal(data)
	if err != nil {
		return nil, err
	}

	return v, nil
}

// CalculateQuorum returns the minimum number of guardians that need to sign a VAA for a given guardian set.
//
// The canonical source is the calculation in the contracts (solana/bridge/src/processor.rs and
// ethereum/contracts/Wormhole.sol), and this needs to match the implementation in the contracts.
func CalculateQuorum(numGuardians int) int {
	return (numGuardians*2)/3 + 1
}

func (k Keeper) VerifyVAA(ctx sdk.Context, vaa *vaa.VAA) error {
	guardianSet, exists := k.GetGuardianSet(ctx, vaa.GuardianSetIndex)
	if !exists {
		return types.ErrGuardianSetNotFound
	}

	if 0 < guardianSet.ExpirationTime && guardianSet.ExpirationTime < uint64(ctx.BlockTime().Unix()) {
		return types.ErrGuardianSetExpired
	}

	// Verify quorum
	quorum := CalculateQuorum(len(guardianSet.Keys))
	if len(vaa.Signatures) < quorum {
		return types.ErrNoQuorum
	}

	// Verify signatures
	ok := vaa.VerifySignatures(guardianSet.KeysAsAddresses())
	if !ok {
		return types.ErrSignaturesInvalid
	}

	return nil
}

// Verify a governance VAA:
// - Check signatures
// - Replay protection
// - Check the source chain and address is governance
// - Check the governance payload is for wormchain and the specified module
// - return the parsed action and governance payload
func (k Keeper) VerifyVAAGovernance(ctx sdk.Context, v *vaa.VAA, module [32]byte) (action byte, payload []byte, err error) {
	if err = k.VerifyVAA(ctx, v); err != nil {
		return
	}
	_, known := k.GetReplayProtection(ctx, v.HexDigest())
	if known {
		err = types.ErrVAAAlreadyExecuted
		return
	}
	// Prevent replay
	k.SetReplayProtection(ctx, types.ReplayProtection{Index: v.HexDigest()})

	config, ok := k.GetConfig(ctx)
	if !ok {
		err = types.ErrNoConfig
		return
	}

	if !bytes.Equal(v.EmitterAddress[:], config.GovernanceEmitter) {
		err = types.ErrInvalidGovernanceEmitter
		return
	}
	if v.EmitterChain != vaa.ChainID(config.GovernanceChain) {
		err = types.ErrInvalidGovernanceEmitter
		return
	}
	if len(v.Payload) < 35 {
		err = types.ErrGovernanceHeaderTooShort
		return
	}

	// Check governance header
	if !bytes.Equal(v.Payload[:32], module[:]) {
		err = types.ErrUnknownGovernanceModule
		return
	}

	// Decode header
	action = v.Payload[32]
	chain := binary.BigEndian.Uint16(v.Payload[33:35])
	payload = v.Payload[35:]

	if chain != 0 && chain != uint16(config.ChainId) {
		err = types.ErrInvalidGovernanceTargetChain
		return
	}

	return
}
