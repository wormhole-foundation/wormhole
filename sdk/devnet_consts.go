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
	vaa.ChainIDAlgorand:  "8edf5b0e108c3a1a0a4b704cc89591f2ad8d50df24e991567e640ed720a94be2",
	vaa.ChainIDWormchain: "0000000000000000000000001711cd63b2c545ee6545415d3cc0bda6425c43c4",
	vaa.ChainIDSui:       "4c6e17eba2ed24d21846217bd689c99bb57eb2a1fbc268840a4a41072548b5ad",
}

// KnownDevnetNFTBridgeEmitters is a map of known NFT emitters used during development.
var KnownDevnetNFTBridgeEmitters = buildEmitterMap(knownDevnetNFTBridgeEmitters)
var knownDevnetNFTBridgeEmitters = map[vaa.ChainID]string{
	vaa.ChainIDSolana:   "96ee982293251b48729804c8e8b24b553eb6b887867024948d2236fd37a577ab",
	vaa.ChainIDEthereum: "00000000000000000000000026b4afb60d6c903165150c6f0aa14f8016be4aec",
	vaa.ChainIDBSC:      "00000000000000000000000026b4afb60d6c903165150c6f0aa14f8016be4aec",
}
