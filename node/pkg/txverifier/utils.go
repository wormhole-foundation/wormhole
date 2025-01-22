package txverifier

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// Constants
const (
	MAX_DECIMALS = 8
	KEY_FORMAT   = "%s-%d"
)

// Extracts the value at the given path from the JSON object, and casts it to
// type T. If the path does not exist in the object, an error is returned.
func extractFromJsonPath[T any](data json.RawMessage, path string) (T, error) {
	var defaultT T

	var obj map[string]interface{}
	err := json.Unmarshal(data, &obj)
	if err != nil {
		return defaultT, err
	}

	// Split the path and iterate over the keys, except for the final key. For
	// each key, check if it exists in the object. If it does exist and is a map,
	// update the object to the value of the key.
	keys := strings.Split(path, ".")
	for _, key := range keys[:len(keys)-1] {
		if obj[key] == nil {
			return defaultT, fmt.Errorf("key %s not found", key)
		}

		if v, ok := obj[key].(map[string]interface{}); ok {
			obj = v
		} else {
			return defaultT, fmt.Errorf("can't convert to key to map[string]interface{} type")
		}
	}

	// If the final key exists in the object, return the value as T. Otherwise,
	// return an error.
	if value, exists := obj[keys[len(keys)-1]]; exists {
		if v, ok := value.(T); ok {
			return v, nil
		} else {
			return defaultT, fmt.Errorf("can't convert to type T")
		}
	} else {
		return defaultT, fmt.Errorf("key %s not found", keys[len(keys)-1])
	}
}

// Normalize the amount to 8 decimals. If the amount has more than 8 decimals,
// the amount is divided by 10^(decimals-8). If the amount has less than 8
// decimals, the amount is returned as is.
// https://wormhole.com/docs/build/start-building/supported-networks/evm/#addresses
func normalize(amount *big.Int, decimals uint8) (normalizedAmount *big.Int) {
	if amount == nil {
		return nil
	}
	if decimals > MAX_DECIMALS {
		exponent := new(big.Int).SetInt64(int64(decimals - 8))
		multiplier := new(big.Int).Exp(new(big.Int).SetInt64(10), exponent, nil)
		normalizedAmount = new(big.Int).Div(amount, multiplier)
	} else {
		return amount
	}

	return normalizedAmount
}

// denormalize() scales an amount to its native decimal representation by multiplying it by some power of 10.
// See also:
//   - documentation:
//     https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0003_token_bridge.md#handling-of-token-amounts-and-decimals
//     https://wormhole.com/docs/build/start-building/supported-networks/evm/#addresses
//   - solidity implementation:
//     https://github.com/wormhole-foundation/wormhole/blob/91ec4d1dc01f8b690f0492815407505fb4587520/ethereum/contracts/bridge/Bridge.sol#L295-L300
func denormalize(
	amount *big.Int,
	decimals uint8,
) (denormalizedAmount *big.Int) {
	if decimals > 8 {
		// Scale from 8 decimals to `decimals`
		exponent := new(big.Int).SetInt64(int64(decimals - 8))
		multiplier := new(big.Int).Exp(new(big.Int).SetInt64(10), exponent, nil)
		denormalizedAmount = new(big.Int).Mul(amount, multiplier)

	} else {
		// No scaling necessary
		denormalizedAmount = new(big.Int).Set(amount)
	}

	return denormalizedAmount
}

// SupportedChains returns a slice of Wormhole Chain IDs that have a Transfer Verifier implementation.
func SupportedChains() []vaa.ChainID {
	return []vaa.ChainID{
		vaa.ChainIDEthereum,
		vaa.ChainIDSui,
	}
}
