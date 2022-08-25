package types

import (
	"encoding/hex"
	"fmt"
	math "math"
	"math/big"
	"strconv"
	"strings"

	whtypes "github.com/certusone/wormhole-chain/x/wormhole/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	btypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Truncate an amount
func Truncate(coin sdk.Coin, meta btypes.Metadata) (normalized sdk.Coin, err error) {
	factor, err := truncFactor(meta)
	if err != nil {
		return normalized, err
	}

	amt := new(big.Int).Div(coin.Amount.BigInt(), factor)
	return sdk.NewCoin(coin.Denom, sdk.NewIntFromBigInt(amt)), nil
}

// Untruncate an amount
func Untruncate(coin sdk.Coin, meta btypes.Metadata) (normalized sdk.Coin, err error) {
	factor, err := truncFactor(meta)
	if err != nil {
		return normalized, err
	}

	amt := new(big.Int).Mul(coin.Amount.BigInt(), factor)
	return sdk.NewCoin(coin.Denom, sdk.NewIntFromBigInt(amt)), nil
}

// Compute truncation factor for a given token meta.
// This is max(1, exponent - 8). If you divide an amount by this number, the
// result will have 8 decimals.
func truncFactor(meta btypes.Metadata) (factor *big.Int, err error) {
	// Find the display denom to figure out decimals
	var displayDenom *btypes.DenomUnit
	for _, denom := range meta.DenomUnits {
		if denom.Denom == meta.Display {
			displayDenom = denom
			break
		}
	}
	if displayDenom == nil {
		return new(big.Int), ErrDisplayUnitNotFound
	}

	if displayDenom.Exponent > math.MaxUint8 {
		return nil, ErrExponentTooLarge
	}

	if displayDenom.Exponent > 8 {
		return new(big.Int).SetInt64(int64(math.Pow10(int(displayDenom.Exponent - 8)))), nil
	} else {
		return big.NewInt(1), nil
	}
}

var uwormChain uint16 = 1
var uwormAddress = [32]byte{0x16, 0x58, 0x09, 0x73, 0x92, 0x40, 0xa0, 0xac, 0x03, 0xb9, 0x84, 0x40, 0xfe, 0x89, 0x85, 0x54, 0x8e, 0x3a, 0xa6, 0x83, 0xcd, 0x0d, 0x4d, 0x9d, 0xf5, 0xb5, 0x65, 0x96, 0x69, 0xfa, 0xa3, 0x01}

func IsWORMToken(tokenChain uint16, tokenAddress [32]byte) bool {
	// TODO(csongor): figure out WORM token address on Solana
	// TODO(csongor): this should be configurable in the genesis
	return tokenChain == uwormChain && tokenAddress == uwormAddress
}

// Derive the name of a wrapped token based on its origin chain and token address
//
// For most tokens, this looks something like
//
//		"wh/00001/165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa300"
//
// where the string "wh" is followed by the token chain (fixed 5 characters)
// in decimal, followed by the token address in hex (fixed 64 character).
//
// For the special case of the wormhole token it returns
//
// 		"uworm"
//
func GetWrappedCoinIdentifier(tokenChain uint16, tokenAddress [32]byte) string {
	if IsWORMToken(tokenChain, tokenAddress) {
		return "uworm"
	} else {
		// log10(2^16) = 5, so we print token chain in 5 decimals
		return fmt.Sprintf("wh/%05d/%064x", tokenChain, tokenAddress)
	}
}

// From a given wrapped token identifier, return the token chain and the token
// address.
func GetWrappedCoinMeta(identifier string) (tokenChain uint16, tokenAddress [32]byte, wrapped bool) {
	if identifier == "uworm" {
		return uwormChain, uwormAddress, true
	}

	parts := strings.Split(identifier, "/")
	if len(parts) != 3 {
		return 0, [32]byte{}, false
	}
	if parts[0] != "wh" && parts[0] != "bwh" {
		return 0, [32]byte{}, false
	}
	if len(parts[1]) != 5 {
		return 0, [32]byte{}, false
	}
	if len(parts[2]) != 64 {
		return 0, [32]byte{}, false
	}

	tokenChain64, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return 0, [32]byte{}, false
	}
	tokenChain = uint16(tokenChain64)

	tokenAddressBytes, err := hex.DecodeString(parts[2])
	if err != nil {
		return 0, [32]byte{}, false
	}
	copy(tokenAddress[:], tokenAddressBytes[:])

	return tokenChain, tokenAddress, true
}

// Get the token chain and address. For wrapped assets, this is the address on
// the original chain, for native assets, it's the (left-padded) display name.
func GetTokenMeta(config whtypes.Config, identifier string) (chainId uint16, tokenAddress [32]byte, err error) {
	tokenChain, tokenAddress, wrapped := GetWrappedCoinMeta(identifier)

	if wrapped {
		return tokenChain, tokenAddress, nil
	} else {
		padded, err := PadStringToByte32(identifier)
		if err != nil {
			return 0, [32]byte{}, err
		}
		return uint16(config.ChainId), padded, nil
	}
}

// PadStringToByte32 left zero pads a string to the ethereum type bytes32
func PadStringToByte32(s string) (padded [32]byte, err error) {
	if len(s) > 32 {
		return [32]byte{}, fmt.Errorf("string is too long; %d > 32", len(s))
	}

	b := []byte(s)

	left := make([]byte, 32-len(s))
	copy(padded[:], append(left[:], b[:]...))
	return padded, nil
}
