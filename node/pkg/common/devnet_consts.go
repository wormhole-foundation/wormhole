package common

import (
	"github.com/certusone/wormhole/node/pkg/vaa"
)

// KnownDevnetTokenbridgeEmitters is a list of known tokenbridge emitters used during development.
var KnownDevnetTokenbridgeEmitters = buildEmitterMap(knownDevnetTokenbridgeEmitters)
var knownDevnetTokenbridgeEmitters = map[vaa.ChainID]string{
	vaa.ChainIDSolana:   "c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f",
	vaa.ChainIDEthereum: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
	vaa.ChainIDTerra:    "000000000000000000000000784999135aaa8a3ca5914468852fdddbddd8789d",
	vaa.ChainIDBSC:      "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
}
