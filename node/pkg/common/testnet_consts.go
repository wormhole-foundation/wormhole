package common

import "github.com/certusone/wormhole/node/pkg/vaa"

// KnownTestnetEmitters is a list of known emitters on the various L1 testnets.
var KnownTestnetEmitters = buildKnownEmitters(knownTestnetTokenbridgeEmitters, knownTestnetNFTBridgeEmitters)

// KnownTestnetTokenbridgeEmitters is a map of known tokenbridge emitters on the various L1 testnets.
var KnownTestnetTokenbridgeEmitters = buildEmitterMap(knownTestnetTokenbridgeEmitters)
var knownTestnetTokenbridgeEmitters = map[vaa.ChainID]string{
	vaa.ChainIDSolana:          "3b26409f8aaded3f5ddca184695aa6a0fa829b0c85caf84856324896d214ca98",
	vaa.ChainIDEthereum:        "000000000000000000000000f890982f9310df57d00f659cf4fd87e65aded8d7",
	vaa.ChainIDTerra:           "0000000000000000000000000c32d68d8f22613f6b9511872dad35a59bfdf7f0",
	vaa.ChainIDTerra2:          "c3d4c6c2bcba163de1defb7e8f505cdb40619eee4fa618678955e8790ae1448d",
	vaa.ChainIDBSC:             "0000000000000000000000009dcf9d205c9de35334d646bee44b2d2859712a09",
	vaa.ChainIDPolygon:         "000000000000000000000000377D55a7928c046E18eEbb61977e714d2a76472a",
	vaa.ChainIDAvalanche:       "00000000000000000000000061e44e506ca5659e6c0bba9b678586fa2d729756",
	vaa.ChainIDOasis:           "00000000000000000000000088d8004a9bdbfd9d28090a02010c19897a29605c",
	vaa.ChainIDAlgorand:        "6241ffdc032b693bfb8544858f0403dec86f2e1720af9f34f8d65fe574b6238c",
	vaa.ChainIDAurora:          "000000000000000000000000d05ed3ad637b890d68a854d607eeaf11af456fba",
	vaa.ChainIDFantom:          "000000000000000000000000599cea2204b4faecd584ab1f2b6aca137a0afbe8",
	vaa.ChainIDKarura:          "000000000000000000000000d11de1f930ea1f7dd0290fe3a2e35b9c91aefb37",
	vaa.ChainIDAcala:           "000000000000000000000000eba00cbe08992edd08ed7793e07ad6063c807004",
	vaa.ChainIDKlaytn:          "000000000000000000000000c7a13be098720840dea132d860fdfa030884b09a",
	vaa.ChainIDCelo:            "00000000000000000000000005ca6037ec51f8b712ed2e6fa72219feae74e153",
	vaa.ChainIDMoonbeam:        "000000000000000000000000bc976d4b9d57e57c3ca52e1fd136c45ff7955a96",
	vaa.ChainIDNeon:            "000000000000000000000000d11de1f930ea1f7dd0290fe3a2e35b9c91aefb37",
	vaa.ChainIDEthereumRopsten: "000000000000000000000000F174F9A837536C449321df1Ca093Bb96948D5386",
}

// KnownTestnetNFTBridgeEmitters is a map  of known NFT emitters on the various L1 testnets.
var KnownTestnetNFTBridgeEmitters = buildEmitterMap(knownTestnetNFTBridgeEmitters)
var knownTestnetNFTBridgeEmitters = map[vaa.ChainID]string{
	vaa.ChainIDSolana:          "752a49814e40b96b097207e4b53fdd330544e1e661653fbad4bc159cc28a839e",
	vaa.ChainIDEthereum:        "000000000000000000000000d8e4c2dbdd2e2bd8f1336ea691dbff6952b1a6eb",
	vaa.ChainIDBSC:             "000000000000000000000000cd16e5613ef35599dc82b24cb45b5a93d779f1ee",
	vaa.ChainIDPolygon:         "00000000000000000000000051a02d0dcb5e52f5b92bdaa38fa013c91c7309a9",
	vaa.ChainIDAvalanche:       "000000000000000000000000d601baf2eee3c028344471684f6b27e789d9075d",
	vaa.ChainIDOasis:           "000000000000000000000000c5c25b41ab0b797571620f5204afa116a44c0eba",
	vaa.ChainIDAurora:          "0000000000000000000000008f399607e9ba2405d87f5f3e1b78d950b44b2e24",
	vaa.ChainIDFantom:          "00000000000000000000000063ed9318628d26bdcb15df58b53bb27231d1b227",
	vaa.ChainIDKarura:          "0000000000000000000000000a693c2d594292b6eb89cb50efe4b0b63dd2760d",
	vaa.ChainIDAcala:           "00000000000000000000000096f1335e0acab3cfd9899b30b2374e25a2148a6e",
	vaa.ChainIDKlaytn:          "00000000000000000000000094c994fc51c13101062958b567e743f1a04432de",
	vaa.ChainIDCelo:            "000000000000000000000000acd8190f647a31e56a656748bc30f69259f245db",
	vaa.ChainIDMoonbeam:        "00000000000000000000000098a0f4b96972b32fcb3bd03caeb66a44a6ab9edb",
	vaa.ChainIDNeon:            "000000000000000000000000a52da3b1ffd258a2ffb7719a6aee24095eee24e2",
	vaa.ChainIDEthereumRopsten: "0000000000000000000000002b048da40f69c8dc386a56705915f8e966fe1eba",
}
