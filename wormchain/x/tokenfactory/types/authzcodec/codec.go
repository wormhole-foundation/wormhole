package authzcodec

// Note: this file is a copy from authz/codec in 0.46 so we can be compatible with 0.45

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var (
	Amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(Amino)
)

func init() {
	// Register all Amino interfaces and concrete types on the authz Amino codec so that this can later be
	// used to properly serialize MsgGrant and MsgExec instances
	sdk.RegisterLegacyAminoCodec(Amino)
	cryptocodec.RegisterCrypto(Amino)
	codec.RegisterEvidences(Amino)

	Amino.Seal()
}
