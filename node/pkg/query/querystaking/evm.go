package querystaking

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

type StakeAndSigner struct {
	StakeInfo *StakeInfo     `abi:"StakeInfo"`
	Signer    common.Address `abi:"Signer"`
}

type StakeInfo struct {
	Amount               *uint256.Int `abi:"uint256"`
	ConversionTableIndex *uint256.Int `abi:"uint256"`
	LockupEnd            uint64       `abi:"uint48"`
	AccessEnd            uint64       `abi:"uint48"`
	LastClaimed          uint64       `abi:"uint48"`
	Capacity             *uint256.Int `abi:"uint256"`
}

// ParseStakeInfo parses raw contract call data into a StakeInfo struct.
func ParseStakeInfo(data []byte) (*StakeInfo, error) {
	expectedLength := 32 * 6

	if len(data) != expectedLength {
		return nil, fmt.Errorf("invalid length: got %d want %d", len(data), expectedLength)
	}

	stakeInfo := &StakeInfo{}

	stakeInfo.Amount = uint256.NewInt(0).SetBytes(data[0:32])
	stakeInfo.ConversionTableIndex = uint256.NewInt(0).SetBytes(data[32:64])

	tmp := uint256.NewInt(0)
	stakeInfo.LockupEnd = tmp.SetBytes(data[64:96]).Uint64()
	stakeInfo.AccessEnd = tmp.SetBytes(data[96:128]).Uint64()
	stakeInfo.LastClaimed = tmp.SetBytes(data[128:160]).Uint64()
	stakeInfo.Capacity = uint256.NewInt(0).SetBytes(data[160:192])

	return stakeInfo, nil
}

// PackStakesCall creates call data for the getStakeInfo(address) function.
func PackStakesCall(staker common.Address) []byte {
	selector := crypto.Keccak256([]byte("getStakeInfo(address)"))[:4]
	paddedAddress := common.LeftPadBytes(staker.Bytes(), 32)

	result := make([]byte, 4+32)
	copy(result[0:4], selector)
	copy(result[4:36], paddedAddress)

	return result
}

// PackStakerSignersCall creates call data for the stakerSigners(address) function.
func PackStakerSignersCall(staker common.Address) []byte {
	selector := crypto.Keccak256([]byte("stakerSigners(address)"))[:4]
	paddedAddress := common.LeftPadBytes(staker.Bytes(), 32)

	result := make([]byte, 4+32)
	copy(result[0:4], selector)
	copy(result[4:36], paddedAddress)

	return result
}

// PackIsBlocklistedCall creates call data for the isBlocklisted(address) function.
func PackIsBlocklistedCall(user common.Address) []byte {
	selector := crypto.Keccak256([]byte("isBlocklisted(address)"))[:4]
	paddedAddress := common.LeftPadBytes(user.Bytes(), 32)

	result := make([]byte, 4+32)
	copy(result[0:4], selector)
	copy(result[4:36], paddedAddress)

	return result
}

// ParseBoolResult parses the result of a bool-returning contract call.
func ParseBoolResult(data []byte) (bool, error) {
	if len(data) != 32 {
		return false, fmt.Errorf("invalid bool data length: got %d want 32", len(data))
	}
	if [31]byte(data[:31]) != [31]byte{} {
		return false, fmt.Errorf("invalid bool data: leading bytes not zero")
	}
	if data[31] > 1 {
		return false, fmt.Errorf("invalid bool data: expected 0 or 1, got %d", data[31])
	}
	// The last byte of the 32-byte result contains the boolean value
	return data[31] == 1, nil
}

// PackQueryTypePoolsCall creates call data for the queryTypePools(bytes32) function.
func PackQueryTypePoolsCall(queryType [32]byte) []byte {
	selector := crypto.Keccak256([]byte("queryTypePools(bytes32)"))[:4]

	result := make([]byte, 4+32)
	copy(result[0:4], selector)
	copy(result[4:36], queryType[:])

	return result
}

// PackConversionTableHistoryCall creates call data for the conversionTableHistory(uint256) function.
func PackConversionTableHistoryCall(index *uint256.Int) []byte {
	selector := crypto.Keccak256([]byte("conversionTableHistory(uint256)"))[:4]
	paddedIndex := common.LeftPadBytes(index.Bytes(), 32)

	result := make([]byte, 4+32)
	copy(result[0:4], selector)
	copy(result[4:36], paddedIndex)

	return result
}

// ParseConversionTableEntry parses the result of conversionTableHistory(uint256) call.
// The conversion table entry is a bytes32 value.
func ParseConversionTableEntry(data []byte) ([32]byte, error) {
	if len(data) != 32 {
		return [32]byte{}, fmt.Errorf("invalid conversion table entry length: got %d want 32", len(data))
	}

	var entry [32]byte
	copy(entry[:], data)
	return entry, nil
}

// PackGetConversionTableHistoryLengthCall creates call data for the getConversionTableHistoryLength() function.
func PackGetConversionTableHistoryLengthCall() []byte {
	selector := crypto.Keccak256([]byte("getConversionTableHistoryLength()"))[:4]
	return selector
}

// ParseUint256Result parses a uint256 result from a contract call.
func ParseUint256Result(data []byte) (*uint256.Int, error) {
	if len(data) != 32 {
		return nil, fmt.Errorf("invalid uint256 data length: got %d want 32", len(data))
	}
	return uint256.NewInt(0).SetBytes(data), nil
}

// PackStakingTokenCall creates call data for the STAKING_TOKEN() function.
func PackStakingTokenCall() []byte {
	selector := crypto.Keccak256([]byte("STAKING_TOKEN()"))[:4]
	return selector
}

// PackDecimalsCall creates call data for the decimals() function on an ERC20 token.
func PackDecimalsCall() []byte {
	selector := crypto.Keccak256([]byte("decimals()"))[:4]
	return selector
}

// ParseAddress parses an address from a contract call.
func ParseAddress(data []byte) (common.Address, error) {
	if len(data) != 32 {
		return common.Address{}, fmt.Errorf("invalid address data length: got %d want 32", len(data))
	}
	if [12]byte(data[:12]) != [12]byte{} {
		return common.Address{}, fmt.Errorf("invalid address data: leading bytes are not zero")
	}
	return common.BytesToAddress(data[12:32]), nil
}

// ParseUint8Result parses a uint8 result from a contract call.
func ParseUint8Result(data []byte) (uint8, error) {
	if len(data) != 32 {
		return 0, fmt.Errorf("invalid uint8 data length: got %d want 32", len(data))
	}
	if [31]byte(data[:31]) != [31]byte{} {
		return 0, fmt.Errorf("invalid uint8 data: leading bytes are not zero")
	}
	return data[31], nil
}

// HasStake returns true if the stake amount is greater than zero.
func (si *StakeInfo) HasStake() bool {
	return si.Amount != nil && si.Amount.Cmp(uint256.NewInt(0)) > 0
}

// HasExpired returns true if the stake has passed or will pass the access period
// within the cache duration. This prevents cached results from granting access
// after expiration. For example, with a 5-minute cache, a stake expiring at time T
// will be considered expired at time (T - cacheDurationSeconds) to ensure the cached
// result doesn't grant access beyond the actual expiration time.
func (si *StakeInfo) HasExpired(currentTimestamp uint64, cacheDurationSeconds uint64) bool {
	// Check if the stake will expire within the cache validity period.
	// This ensures we don't cache a "valid" result that will become invalid
	// before the cache expires.
	return currentTimestamp+cacheDurationSeconds >= si.AccessEnd
}
