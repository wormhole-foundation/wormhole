package common

import (
	"fmt"

	"github.com/certusone/wormhole/node/pkg/vaa"
)

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

type (
	EmitterType uint8
)

const (
	EmitterTypeUnset   EmitterType = 0
	EmitterCoreBridge  EmitterType = 1
	EmitterTokenBridge EmitterType = 2
	EmitterNFTBridge   EmitterType = 3
)

func (et EmitterType) String() string {
	switch et {
	case EmitterTypeUnset:
		return "unset"
	case EmitterCoreBridge:
		return "Core"
	case EmitterTokenBridge:
		return "TokenBridge"
	case EmitterNFTBridge:
		return "NFTBridge"
	default:
		return fmt.Sprintf("unknown emitter type: %d", et)
	}
}

// KnownEmitters is a list of well-known mainnet emitters we want to take into account
// when iterating over all emitters - like for finding and repairing missing messages.
//
// Wormhole is not permissioned - anyone can use it. Adding contracts to this list is
// entirely optional and at the core team's discretion.
//
var KnownEmitters = []struct {
	ChainID    vaa.ChainID
	Emitter    string
	BridgeType EmitterType
}{
	{vaa.ChainIDSolana, "ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5", EmitterTokenBridge},
	{vaa.ChainIDSolana, "0def15a24423e1edd1a5ab16f557b9060303ddbab8c803d2ee48f4b78a1cfd6b", EmitterNFTBridge},
	{vaa.ChainIDEthereum, "0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585", EmitterTokenBridge},
	{vaa.ChainIDEthereum, "0000000000000000000000006ffd7ede62328b3af38fcd61461bbfc52f5651fe", EmitterNFTBridge},
	{vaa.ChainIDTerra, "0000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2", EmitterTokenBridge},
	{vaa.ChainIDTerra2, "a463ad028fb79679cfc8ce1efba35ac0e77b35080a1abe9bebe83461f176b0a3", EmitterTokenBridge},
	{vaa.ChainIDBSC, "000000000000000000000000b6f6d86a8f9879a9c87f643768d9efc38c1da6e7", EmitterTokenBridge},
	{vaa.ChainIDBSC, "0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde", EmitterNFTBridge},
	{vaa.ChainIDPolygon, "0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde", EmitterTokenBridge},
	{vaa.ChainIDPolygon, "00000000000000000000000090bbd86a6fe93d3bc3ed6335935447e75fab7fcf", EmitterNFTBridge},
	{vaa.ChainIDAvalanche, "0000000000000000000000000e082f06ff657d94310cb8ce8b0d9a04541d8052", EmitterTokenBridge},
	{vaa.ChainIDAvalanche, "000000000000000000000000f7b6737ca9c4e08ae573f75a97b73d7a813f5de5", EmitterNFTBridge},
	{vaa.ChainIDOasis, "0000000000000000000000005848c791e09901b40a9ef749f2a6735b418d7564", EmitterTokenBridge},
	{vaa.ChainIDOasis, "00000000000000000000000004952d522ff217f40b5ef3cbf659eca7b952a6c1", EmitterNFTBridge},
	{vaa.ChainIDAurora, "00000000000000000000000051b5123a7b0F9b2bA265f9c4C8de7D78D52f510F", EmitterTokenBridge},
	{vaa.ChainIDAurora, "0000000000000000000000006dcC0484472523ed9Cdc017F711Bcbf909789284", EmitterNFTBridge},
	{vaa.ChainIDFantom, "0000000000000000000000007C9Fc5741288cDFdD83CeB07f3ea7e22618D79D2", EmitterTokenBridge},
	{vaa.ChainIDFantom, "000000000000000000000000A9c7119aBDa80d4a4E0C06C8F4d8cF5893234535", EmitterNFTBridge},
	{vaa.ChainIDKarura, "000000000000000000000000ae9d7fe007b3327AA64A32824Aaac52C42a6E624", EmitterTokenBridge},
	{vaa.ChainIDKarura, "000000000000000000000000b91e3638F82A1fACb28690b37e3aAE45d2c33808", EmitterNFTBridge},
	{vaa.ChainIDAcala, "000000000000000000000000ae9d7fe007b3327AA64A32824Aaac52C42a6E624", EmitterTokenBridge},
	{vaa.ChainIDAcala, "000000000000000000000000b91e3638F82A1fACb28690b37e3aAE45d2c33808", EmitterNFTBridge},
	{vaa.ChainIDKlaytn, "0000000000000000000000005b08ac39EAED75c0439FC750d9FE7E1F9dD0193F", EmitterTokenBridge},
	{vaa.ChainIDKlaytn, "0000000000000000000000003c3c561757BAa0b78c5C025CdEAa4ee24C1dFfEf", EmitterNFTBridge},
	{vaa.ChainIDCelo, "000000000000000000000000796Dff6D74F3E27060B71255Fe517BFb23C93eed", EmitterTokenBridge},
	{vaa.ChainIDCelo, "000000000000000000000000A6A377d75ca5c9052c9a77ED1e865Cc25Bd97bf3", EmitterNFTBridge},
}
