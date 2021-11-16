package keeper

import (
	"github.com/certusone/wormhole-chain/x/wormhole/types"
	"github.com/certusone/wormhole/node/pkg/vaa"
	sdk "github.com/cosmos/cosmos-sdk/types"
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
