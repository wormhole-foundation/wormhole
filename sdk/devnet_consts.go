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
	vaa.ChainIDTerra:     "000000000000000000000000784999135aaa8a3ca5914468852fdddbddd8789d",
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
