package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

// Parameter store keys.
var (
	KeyDenomCreationFee        = []byte("DenomCreationFee")
	KeyDenomCreationGasConsume = []byte("DenomCreationGasConsume")
)

// ParamTable for tokenfactory module.
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

func NewParams(denomCreationFee sdk.Coins) Params {
	return Params{
		DenomCreationFee: denomCreationFee,
	}
}

// default tokenfactory module parameters.
func DefaultParams() Params {
	return Params{
		DenomCreationFee: nil,
		//DenomCreationFee:        sdk.NewCoins(sdk.NewInt64Coin(sdk.DefaultBondDenom, 10_000_000)),
		DenomCreationGasConsume: 0,
	}
}

// validate params.
func (p Params) Validate() error {
	err := validateDenomCreationFee(p.DenomCreationFee)

	return err
}

// Implements params.ParamSet.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(KeyDenomCreationFee, &p.DenomCreationFee, validateDenomCreationFee),
		paramtypes.NewParamSetPair(KeyDenomCreationGasConsume, &p.DenomCreationGasConsume, validateDenomCreationFeeGasConsume),
	}
}

func validateDenomCreationFee(i interface{}) error {
	v, ok := i.(sdk.Coins)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.Validate() != nil {
		return fmt.Errorf("invalid denom creation fee: %+v", i)
	}

	return nil
}

func validateDenomCreationFeeGasConsume(i interface{}) error {
	_, ok := i.(uint64)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	return nil
}
