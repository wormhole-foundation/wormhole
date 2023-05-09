package vaa

// CalculateQuorum returns the minimum number of guardians that need to sign a VAA for a given guardian set.
//
// The canonical source is the calculation in the contracts (solana/bridge/src/processor.rs and
// ethereum/contracts/Wormhole.sol), and this needs to match the implementation in the contracts.
func CalculateQuorum(numGuardians int) int {
	if numGuardians < 0 {
		panic("Invalid numGuardians is less than zero")
	}
	return ((numGuardians * 2) / 3) + 1
}
