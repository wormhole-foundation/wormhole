package common

import "github.com/certusone/wormhole/node/pkg/vaa"

// PublicRPCEndpoints is a list of known public RPC endpoints for mainnet, operated by
// Wormhole guardian nodes.
//
// This list is duplicated a couple times across the codebase - make to to update all copies!
//
var PublicRPCEndpoints = []string{
	"https://wormhole-v2-mainnet-api.certus.one",
	"https://wormhole.inotel.ro",
	"https://wormhole-v2-mainnet-api.mcf.rocks",
	"https://wormhole-v2-mainnet-api.chainlayer.network",
	"https://wormhole-v2-mainnet-api.staking.fund",
	"https://wormhole-v2-mainnet.01node.com",
}

// KnownEmitters is a list of well-known mainnet emitters we want to take into account
// when iterating over all emitters - like for finding and repairing missing messages.
//
// Wormhole is not permissioned - anyone can use it. Adding contracts to this list is
// entirely optional and at the core team's discretion.
//
var KnownEmitters = []struct {
	ChainID vaa.ChainID
	Emitter string
}{
	{vaa.ChainIDSolana, "ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5"},    // 1 Solana Token Bridge
	{vaa.ChainIDSolana, "0def15a24423e1edd1a5ab16f557b9060303ddbab8c803d2ee48f4b78a1cfd6b"},    // 1 Solana NFT Bridge
	{vaa.ChainIDEthereum, "0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585"},  // 2 Eth Token Bridge
	{vaa.ChainIDEthereum, "0000000000000000000000006ffd7ede62328b3af38fcd61461bbfc52f5651fe"},  // 2 Eth NFT Bridge
	{vaa.ChainIDTerra, "0000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2"},     // 3 Terra Token Bridge
	{vaa.ChainIDBSC, "000000000000000000000000b6f6d86a8f9879a9c87f643768d9efc38c1da6e7"},       // 4 BSC Token Bridge
	{vaa.ChainIDBSC, "0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde"},       // 4 BSC NFT Bridge
	{vaa.ChainIDPolygon, "0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde"},   // 5 Polygon Token Bridge
	{vaa.ChainIDPolygon, "00000000000000000000000090bbd86a6fe93d3bc3ed6335935447e75fab7fcf"},   // 5 Polygon NFT Bridge
	{vaa.ChainIDAvalanche, "0000000000000000000000000e082f06ff657d94310cb8ce8b0d9a04541d8052"}, // 6 Avalanche Token Bridge
	{vaa.ChainIDAvalanche, "000000000000000000000000f7b6737ca9c4e08ae573f75a97b73d7a813f5de5"}, // 6 Avalanche NFT Bridge
	{vaa.ChainIDOasis, "0000000000000000000000005848c791e09901b40a9ef749f2a6735b418d7564"},     // 7 Oasis Token Bridge
	{vaa.ChainIDOasis, "00000000000000000000000004952d522ff217f40b5ef3cbf659eca7b952a6c1"},     // 7 Oasis NFT Bridge
}
