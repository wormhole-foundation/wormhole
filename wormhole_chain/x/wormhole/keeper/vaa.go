package keeper

import (
	"github.com/certusone/wormhole-chain/x/wormhole/types"
	"github.com/certusone/wormhole/node/pkg/vaa"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func ParseVAA(data []byte) (*vaa.VAA, error) {
	v, err := vaa.Unmarshal(data)
	if err != nil {
		return nil, types.ErrVAAUnmarshal
	}

	return v, nil
}

// CalculateQuorum returns the minimum number of guardians that need to sign a VAA for a given guardian set.
//
// The canonical source is the calculation in the contracts (solana/bridge/src/processor.rs and
// ethereum/contracts/Wormhole.sol), and this needs to match the implementation in the contracts.
func CalculateQuorum(numGuardians int) int {
	return ((numGuardians*10/3)*2)/10 + 1
}

func (k Keeper) VerifyVAA(ctx sdk.Context, vaa *vaa.VAA) error {
	// TODO(csongor): shouldn't we check guardian set expiry date here? (yes, we should)
	guardianSet, exists := k.GetGuardianSet(ctx, vaa.GuardianSetIndex)
	if !exists {
		return types.ErrGuardianSetNotFound
	}

	// Verify signatures
	ok := vaa.VerifySignatures(guardianSet.KeysAsAddresses())
	if !ok {
		return types.ErrSignaturesInvalid
	}

	// Verify Quorum
	// TODO(csongor): maybe this check should happen before signature
	// verification as it's cheaper
	quorum := CalculateQuorum(len(guardianSet.Keys))
	if len(vaa.Signatures) < quorum {
		return types.ErrNoQuorum
	}

	return nil
}
