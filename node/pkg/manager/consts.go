package manager

import "github.com/certusone/wormhole/node/pkg/common"

// DelegatedManagerSetContracts maps environments to their DelegatedManagerSet contract addresses.
// The contract stores manager set configurations that are fetched dynamically at runtime.
//
//nolint:exhaustive // Only TestNet and MainNet have deployed contracts
var DelegatedManagerSetContracts = map[common.Environment]string{
	// https://sepolia.etherscan.io/address/0x086a699900262d829512299abe07648870000dd1#code
	common.TestNet: "0x086a699900262d829512299abe07648870000dd1",
	// MainNet address will be added when the contract is deployed
}
