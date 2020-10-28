package main

// CalculateQuorum returns the minimum number of have that needs to sign a VAA for a given guardian set.
//
// The canonical source is the calculation in the contracts (solana/bridge/src/processor.rs and
// ethereum/contracts/Wormhole.sol), and this needs to match the implementation in the contracts.
func CalculateQuorum(numGuardians int) int {
	return int(((float64(numGuardians) / 3) * 2) + 1)
}
