package sdk

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// KnownDevnetEmitters is a list of known emitters used during development.
var KnownDevnetEmitters = buildKnownEmitters(knownDevnetTokenbridgeEmitters, knownDevnetNFTBridgeEmitters)

// KnownDevnetTokenbridgeEmitters is a map of known tokenbridge emitters used during development.
var KnownDevnetTokenbridgeEmitters = buildEmitterMap(knownDevnetTokenbridgeEmitters)
var knownDevnetTokenbridgeEmitters = map[vaa.ChainID]string{
	vaa.ChainIDSolana:    "c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f",
	vaa.ChainIDEthereum:  "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
	vaa.ChainIDTerra:     "9e28beafa966b2407bffb0d48651e94972a56e69f3c0897d9e8facbdaeb98386",
	vaa.ChainIDBSC:       "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
	vaa.ChainIDAlgorand:  "8ec299cb7f3efec28f542397e07f07118d74c875f85409ed8e6b93c17b60e992",
	vaa.ChainIDWormchain: "c9138c6e5bd7a2ab79c1a87486c9d7349d064b35ac9f7498f3b207b3a61e6013",
	vaa.ChainIDSui:       "8c6ba6a65f1b9c7fba4c5ad710086ace208e9ac21786a923425efc8167a419f0",
}

// KnownDevnetNFTBridgeEmitters is a map of known NFT emitters used during development.
var KnownDevnetNFTBridgeEmitters = buildEmitterMap(knownDevnetNFTBridgeEmitters)
var knownDevnetNFTBridgeEmitters = map[vaa.ChainID]string{
	vaa.ChainIDSolana:   "96ee982293251b48729804c8e8b24b553eb6b887867024948d2236fd37a577ab",
	vaa.ChainIDEthereum: "00000000000000000000000026b4afb60d6c903165150c6f0aa14f8016be4aec",
	vaa.ChainIDBSC:      "00000000000000000000000026b4afb60d6c903165150c6f0aa14f8016be4aec",
}

// KnownDevnetAutomaticRelayerEmitters is a list of well-known devnet emitters for the Automatic Relayers.
// It is based on this: https://github.com/wormhole-foundation/wormhole/blob/2c9703670eadc48a7dc8967e81ed2823affcc679/sdk/js/src/relayer/consts.ts#L82
// Note that the format of this is different from the other maps because we don't want to limit it to one per chain.
var KnownDevnetAutomaticRelayerEmitters = []struct {
	ChainId vaa.ChainID
	Addr    string
}{
	{ChainId: vaa.ChainIDEthereum, Addr: "000000000000000000000000b98F46E96cb1F519C333FdFB5CCe0B13E0300ED4"},
	{ChainId: vaa.ChainIDBSC, Addr: "000000000000000000000000b98F46E96cb1F519C333FdFB5CCe0B13E0300ED4"},

	// NTT end to end testing uses special emitters in local dev and CI.
	{ChainId: vaa.ChainIDEthereum, Addr: "000000000000000000000000cc680d088586c09c3e0e099a676fa4b6e42467b4"},
	{ChainId: vaa.ChainIDBSC, Addr: "000000000000000000000000cc680d088586c09c3e0e099a676fa4b6e42467b4"},
}

// KnownDevnetWrappedNativeAddress is a map of wrapped native addresses by chain ID, e.g. WETH for Ethereum
var KnownDevnetWrappedNativeAddresses = map[vaa.ChainID]string{
	// WETH deployed by the Tilt devnet configuration.
	vaa.ChainIDEthereum: "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E",
}
