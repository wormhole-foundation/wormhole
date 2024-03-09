package vaa

// CalculateQuorum returns the minimum number of guardians that need to sign a VAA for a given guardian set.
//
// The canonical source is the calculation in the contracts (solana/bridge/src/processor.rs and
// ethereum/contracts/Wormhole.sol), and this needs to match the implementation in the contracts.
func CalculateQuorum(numGuardians int) int {
	// A safety check to avoid caller from ever supplying a negative
	// number, because we're dealing with signed integers
	if numGuardians < 0 {
		panic("Invalid numGuardians is less than zero")
	}

	// The goal here is to achieve a 2/3 quorum, but since we're
	// dividing on int, we need to +1 to avoid the rounding down
	// effect of integer division
	//
	// For example sake, 5 / 2 == 2, but really that's not an
	//   effective 2/3 quorum, so we add 1 for safety to get to 3
	//
	return ((numGuardians * 2) / 3) + 1
}
