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

// KnownDevnetCoreContracts is a map of known core contract addresses used during development.
var KnownDevnetCoreContracts = map[vaa.ChainID]string{
	vaa.ChainIDEthereum: "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550",
}

// KnownDevnetManagerEmitters is a list of known manager emitters used during development.
// Note that the format allows multiple emitters per chain.
var KnownDevnetManagerEmitters = []struct {
	ChainId vaa.ChainID
	Addr    string
}{
	{ChainId: vaa.ChainIDSolana, Addr: "af528793be84ee2c922e2b27b7cae282a4d098f4fe528f35ded8c8c06d0b1090"},
	{ChainId: vaa.ChainIDEthereum, Addr: "00000000000000000000000090F8bf6A479f320ead074411a4B0e7944Ea8c9C1"},
}

// TODO: This should be loaded from the delegated manager set contract instead.
// TODO: This should be listed per chain and per index.
// KnownDevnetManagerSet is the initial delegated manager set for devnet.
// It is a 5-of-7 multisig using the first 7 devnet guardian keys.
// Generated from scripts/generate-manager-set using devnetGuardians from scripts/devnet-consts.json.
var KnownDevnetManagerSet = struct {
	M          uint8
	N          uint8
	PublicKeys [][33]byte
}{
	M: 5,
	N: 7,
	PublicKeys: [][33]byte{
		// guardian-0: 02d4a4629979f0c9fa0f0bb54edf33f87c8c5a1f42c0350a30d68f7e967023e34e
		{0x02, 0xD4, 0xA4, 0x62, 0x99, 0x79, 0xF0, 0xC9, 0xFA, 0x0F, 0x0B, 0xB5, 0x4E, 0xDF, 0x33, 0xF8,
			0x7C, 0x8C, 0x5A, 0x1F, 0x42, 0xC0, 0x35, 0x0A, 0x30, 0xD6, 0x8F, 0x7E, 0x96, 0x70, 0x23, 0xE3,
			0x4E},
		// guardian-1: 03de9544a079988df10b0c847a401862b62d8074b02b994ed5a4f96e6078ee048b
		{0x03, 0xDE, 0x95, 0x44, 0xA0, 0x79, 0x98, 0x8D, 0xF1, 0x0B, 0x0C, 0x84, 0x7A, 0x40, 0x18, 0x62,
			0xB6, 0x2D, 0x80, 0x74, 0xB0, 0x2B, 0x99, 0x4E, 0xD5, 0xA4, 0xF9, 0x6E, 0x60, 0x78, 0xEE, 0x04,
			0x8B},
		// guardian-2: 02d0ddfb81eed4d5ccbb879285c9e52641ca72cc9e89e09b4236a1530abce2d73c
		{0x02, 0xD0, 0xDD, 0xFB, 0x81, 0xEE, 0xD4, 0xD5, 0xCC, 0xBB, 0x87, 0x92, 0x85, 0xC9, 0xE5, 0x26,
			0x41, 0xCA, 0x72, 0xCC, 0x9E, 0x89, 0xE0, 0x9B, 0x42, 0x36, 0xA1, 0x53, 0x0A, 0xBC, 0xE2, 0xD7,
			0x3C},
		// guardian-3: 0312081327a05666bb31510a6930b05d9b103dc36f47bc5bbd858162e0bdfdfc7f
		{0x03, 0x12, 0x08, 0x13, 0x27, 0xA0, 0x56, 0x66, 0xBB, 0x31, 0x51, 0x0A, 0x69, 0x30, 0xB0, 0x5D,
			0x9B, 0x10, 0x3D, 0xC3, 0x6F, 0x47, 0xBC, 0x5B, 0xBD, 0x85, 0x81, 0x62, 0xE0, 0xBD, 0xFD, 0xFC,
			0x7F},
		// guardian-4: 023ab8104c4ba4b5edd183a809831b1ead4f540ab623281012928bc7b26cd1d343
		{0x02, 0x3A, 0xB8, 0x10, 0x4C, 0x4B, 0xA4, 0xB5, 0xED, 0xD1, 0x83, 0xA8, 0x09, 0x83, 0x1B, 0x1E,
			0xAD, 0x4F, 0x54, 0x0A, 0xB6, 0x23, 0x28, 0x10, 0x12, 0x92, 0x8B, 0xC7, 0xB2, 0x6C, 0xD1, 0xD3,
			0x43},
		// guardian-5: 0247541e77c1ffbd34ed666a2d86aa84c372d54c3e611ba2bccbcfadb5faac5c16
		{0x02, 0x47, 0x54, 0x1E, 0x77, 0xC1, 0xFF, 0xBD, 0x34, 0xED, 0x66, 0x6A, 0x2D, 0x86, 0xAA, 0x84,
			0xC3, 0x72, 0xD5, 0x4C, 0x3E, 0x61, 0x1B, 0xA2, 0xBC, 0xCB, 0xCF, 0xAD, 0xB5, 0xFA, 0xAC, 0x5C,
			0x16},
		// guardian-6: 02bac04f860a2287402a36daa71eeeeb46188766f2463a56c684171bd5edfbd7ba
		{0x02, 0xBA, 0xC0, 0x4F, 0x86, 0x0A, 0x22, 0x87, 0x40, 0x2A, 0x36, 0xDA, 0xA7, 0x1E, 0xEE, 0xEB,
			0x46, 0x18, 0x87, 0x66, 0xF2, 0x46, 0x3A, 0x56, 0xC6, 0x84, 0x17, 0x1B, 0xD5, 0xED, 0xFB, 0xD7,
			0xBA},
	},
}
