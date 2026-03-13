package manager

import "github.com/certusone/wormhole/node/pkg/common"

// DelegatedManagerSetContracts maps environments to their DelegatedManagerSet contract addresses.
// The contract stores manager set configurations that are fetched dynamically at runtime.
//
//nolint:exhaustive // MainNet address will be added when the contract is deployed
var DelegatedManagerSetContracts = map[common.Environment]string{
	common.UnsafeDevNet: "0x63B86f40cF28141e82B952c41b43Ba724722a5BD",
	// https://sepolia.etherscan.io/address/0x086a699900262d829512299abe07648870000dd1#code
	common.TestNet: "0x086a699900262d829512299abe07648870000dd1",
}
