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
	vaa.ChainIDWormchain: "45dbea4617971d93188eda21530bc6503d153313b6f575048c2c35dbc6e4fb06",
	vaa.ChainIDSui:       "be8d2e6809d4873bcf1d8be6af2b92500091ad6aa5dc76bc717af86a58d300ca",
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
	{ChainId: vaa.ChainIDEthereum, Addr: "000000000000000000000000E66C1Bc1b369EF4F376b84373E3Aa004E8F4C083"},
	{ChainId: vaa.ChainIDBSC, Addr: "000000000000000000000000E66C1Bc1b369EF4F376b84373E3Aa004E8F4C083"},

	// NTT end to end testing uses special emitters in local dev and CI.
	{ChainId: vaa.ChainIDEthereum, Addr: "00000000000000000000000053855d4b64e9a3cf59a84bc768ada716b5536bc5"},
	{ChainId: vaa.ChainIDEthereum, Addr: "000000000000000000000000E66C1Bc1b369EF4F376b84373E3Aa004E8F4C083"},
	{ChainId: vaa.ChainIDBSC, Addr: "00000000000000000000000053855d4b64e9a3cf59a84bc768ada716b5536bc5"},
	{ChainId: vaa.ChainIDBSC, Addr: "000000000000000000000000E66C1Bc1b369EF4F376b84373E3Aa004E8F4C083"},
}
