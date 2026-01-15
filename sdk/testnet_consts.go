package sdk

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// KnownTestnetEmitters is a list of known emitters on the various L1 testnets.
var KnownTestnetEmitters = buildKnownEmitters(knownTestnetTokenbridgeEmitters, knownTestnetNFTBridgeEmitters)

// KnownTestnetTokenbridgeEmitters is a map of known tokenbridge emitters on the various L1 testnets.
var KnownTestnetTokenbridgeEmitters = buildEmitterMap(knownTestnetTokenbridgeEmitters)
var knownTestnetTokenbridgeEmitters = map[vaa.ChainID]string{
	vaa.ChainIDSolana:          "3b26409f8aaded3f5ddca184695aa6a0fa829b0c85caf84856324896d214ca98",
	vaa.ChainIDEthereum:        "000000000000000000000000f890982f9310df57d00f659cf4fd87e65aded8d7",
	vaa.ChainIDBSC:             "0000000000000000000000009dcf9d205c9de35334d646bee44b2d2859712a09",
	vaa.ChainIDPolygon:         "000000000000000000000000377D55a7928c046E18eEbb61977e714d2a76472a",
	vaa.ChainIDAvalanche:       "00000000000000000000000061e44e506ca5659e6c0bba9b678586fa2d729756",
	vaa.ChainIDAlgorand:        "6241ffdc032b693bfb8544858f0403dec86f2e1720af9f34f8d65fe574b6238c",
	vaa.ChainIDAptos:           "0000000000000000000000000000000000000000000000000000000000000001",
	vaa.ChainIDFantom:          "000000000000000000000000599cea2204b4faecd584ab1f2b6aca137a0afbe8",
	vaa.ChainIDKlaytn:          "000000000000000000000000c7a13be098720840dea132d860fdfa030884b09a",
	vaa.ChainIDCelo:            "00000000000000000000000005ca6037ec51f8b712ed2e6fa72219feae74e153",
	vaa.ChainIDNear:            "c2c0b6ecbbe9ecf91b2b7999f0264018ba68126c2e83bf413f59f712f3a1df55",
	vaa.ChainIDMoonbeam:        "000000000000000000000000bc976d4b9d57e57c3ca52e1fd136c45ff7955a96",
	vaa.ChainIDArbitrum:        "00000000000000000000000023908A62110e21C04F3A4e011d24F901F911744A",
	vaa.ChainIDOptimism:        "000000000000000000000000C7A204bDBFe983FCD8d8E61D02b475D4073fF97e",
	vaa.ChainIDInjective:       "00000000000000000000000003f3e7b2e363f51cf6e57ef85f43a2b91dbce501",
	vaa.ChainIDSui:             "40440411a170b4842ae7dee4f4a7b7a58bc0a98566e998850a7bb87bf5dc05b9",
	vaa.ChainIDBase:            "000000000000000000000000A31aa3FDb7aF7Db93d18DDA4e19F811342EDF780",
	vaa.ChainIDSei:             "9328673cb5de3fd99974cefbbd90fea033f4c59a572abfd7e1a4eebcc5d18157",
	vaa.ChainIDScroll:          "00000000000000000000000022427d90B7dA3fA4642F7025A854c7254E4e45BF",
	vaa.ChainIDMantle:          "00000000000000000000000075Bfa155a9D7A3714b0861c8a8aF0C4633c45b5D",
	vaa.ChainIDMovement:        "0000000000000000000000000000000000000000000000000000000000000002",
	vaa.ChainIDXLayer:          "000000000000000000000000dA91a06299BBF302091B053c6B9EF86Eff0f930D",
	vaa.ChainIDLinea:           "000000000000000000000000C7A204bDBFe983FCD8d8E61D02b475D4073fF97e",
	vaa.ChainIDBerachain:       "000000000000000000000000a10f2eF61dE1f19f586ab8B6F2EbA89bACE63F7a",
	vaa.ChainIDSeiEVM:          "00000000000000000000000023908A62110e21C04F3A4e011d24F901F911744A",
	vaa.ChainIDUnichain:        "000000000000000000000000a10f2eF61dE1f19f586ab8B6F2EbA89bACE63F7a",
	vaa.ChainIDWorldchain:      "000000000000000000000000430855B4D43b8AEB9D2B9869B74d58dda79C0dB2",
	vaa.ChainIDInk:             "000000000000000000000000376428e7f26D5867e69201b275553C45B09EE090",
	vaa.ChainIDHyperEVM:        "0000000000000000000000004a8bc80Ed5a4067f1CCf107057b8270E0cC11A78",
	vaa.ChainIDMezo:            "000000000000000000000000A31aa3FDb7aF7Db93d18DDA4e19F811342EDF780",
	vaa.ChainIDSepolia:         "000000000000000000000000DB5492265f6038831E89f495670FF909aDe94bd9",
	vaa.ChainIDHolesky:         "00000000000000000000000076d093BbaE4529a342080546cAFEec4AcbA59EC6",
	vaa.ChainIDArbitrumSepolia: "000000000000000000000000C7A204bDBFe983FCD8d8E61D02b475D4073fF97e",
	vaa.ChainIDBaseSepolia:     "00000000000000000000000086F55A04690fd7815A3D802bD587e83eA888B239",
	vaa.ChainIDOptimismSepolia: "00000000000000000000000099737Ec4B815d816c49A385943baf0380e75c0Ac",
	vaa.ChainIDPolygonSepolia:  "000000000000000000000000C7A204bDBFe983FCD8d8E61D02b475D4073fF97e",
	vaa.ChainIDWormchain:       "ef5251ea1e99ae48732800ccc7b83b57881232a73eb796b63b1d86ed2ea44e27",
	vaa.ChainIDXRPLEVM:         "0000000000000000000000007d8eBc211C4221eA18E511E4f0fD50c5A539f275",
	vaa.ChainIDMoca:            "000000000000000000000000F97B81E513f53c7a6B57Bd0b103a6c295b3096C5",
	vaa.ChainIDMegaETH:         "0000000000000000000000003D5c2c2BEA15Af5D45F084834c535628C48c42A4",
	vaa.ChainIDMonadTestnet:    "000000000000000000000000F97B81E513f53c7a6B57Bd0b103a6c295b3096C5",
	vaa.ChainIDZeroGravity:     "0000000000000000000000007d8eBc211C4221eA18E511E4f0fD50c5A539f275",
}

// KnownTestnetNFTBridgeEmitters is a map  of known NFT emitters on the various L1 testnets.
var KnownTestnetNFTBridgeEmitters = buildEmitterMap(knownTestnetNFTBridgeEmitters)
var knownTestnetNFTBridgeEmitters = map[vaa.ChainID]string{
	vaa.ChainIDSolana:          "752a49814e40b96b097207e4b53fdd330544e1e661653fbad4bc159cc28a839e",
	vaa.ChainIDEthereum:        "000000000000000000000000d8e4c2dbdd2e2bd8f1336ea691dbff6952b1a6eb",
	vaa.ChainIDBSC:             "000000000000000000000000cd16e5613ef35599dc82b24cb45b5a93d779f1ee",
	vaa.ChainIDPolygon:         "00000000000000000000000051a02d0dcb5e52f5b92bdaa38fa013c91c7309a9",
	vaa.ChainIDAvalanche:       "000000000000000000000000d601baf2eee3c028344471684f6b27e789d9075d",
	vaa.ChainIDFantom:          "00000000000000000000000063ed9318628d26bdcb15df58b53bb27231d1b227",
	vaa.ChainIDKlaytn:          "00000000000000000000000094c994fc51c13101062958b567e743f1a04432de",
	vaa.ChainIDCelo:            "000000000000000000000000acd8190f647a31e56a656748bc30f69259f245db",
	vaa.ChainIDMoonbeam:        "00000000000000000000000098a0f4b96972b32fcb3bd03caeb66a44a6ab9edb",
	vaa.ChainIDArbitrum:        "000000000000000000000000Ee3dB83916Ccdc3593b734F7F2d16D630F39F1D0",
	vaa.ChainIDOptimism:        "00000000000000000000000023908A62110e21C04F3A4e011d24F901F911744A",
	vaa.ChainIDBase:            "000000000000000000000000F681d1cc5F25a3694E348e7975d7564Aa581db59",
	vaa.ChainIDSepolia:         "0000000000000000000000006a0B52ac198e4870e5F3797d5B403838a5bbFD99",
	vaa.ChainIDHolesky:         "000000000000000000000000c8941d483c45eF8FB72E4d1F9dDE089C95fF8171",
	vaa.ChainIDArbitrumSepolia: "00000000000000000000000023908A62110e21C04F3A4e011d24F901F911744A",
	vaa.ChainIDBaseSepolia:     "000000000000000000000000268557122Ffd64c85750d630b716471118F323c8",
	vaa.ChainIDOptimismSepolia: "00000000000000000000000027812285fbe85BA1DF242929B906B31EE3dd1b9f",
	vaa.ChainIDPolygonSepolia:  "00000000000000000000000023908A62110e21C04F3A4e011d24F901F911744A",
}

// KnownTestnetAutomaticRelayerEmitters is a list of well-known testnet emitters for the Automatic Relayers.
// It is based on this: https://github.com/wormhole-foundation/wormhole/blob/2c9703670eadc48a7dc8967e81ed2823affcc679/sdk/js/src/relayer/consts.ts#L14
// Note that the format of this is different from the other maps because we don't want to limit it to one per chain.
var KnownTestnetAutomaticRelayerEmitters = []struct {
	ChainId vaa.ChainID
	Addr    string
}{
	{ChainId: vaa.ChainIDEthereum, Addr: "00000000000000000000000028D8F1Be96f97C1387e94A53e00eCcFb4E75175a"},
	{ChainId: vaa.ChainIDBSC, Addr: "00000000000000000000000080aC94316391752A193C1c47E27D382b507c93F3"},
	{ChainId: vaa.ChainIDPolygon, Addr: "0000000000000000000000000591C25ebd0580E0d4F27A82Fc2e24E7489CB5e0"},
	{ChainId: vaa.ChainIDAvalanche, Addr: "000000000000000000000000A3cF45939bD6260bcFe3D66bc73d60f19e49a8BB"},
	{ChainId: vaa.ChainIDCelo, Addr: "000000000000000000000000306B68267Deb7c5DfCDa3619E22E9Ca39C374f84"},
	{ChainId: vaa.ChainIDMoonbeam, Addr: "0000000000000000000000000591C25ebd0580E0d4F27A82Fc2e24E7489CB5e0"},
	{ChainId: vaa.ChainIDArbitrum, Addr: "000000000000000000000000Ad753479354283eEE1b86c9470c84D42f229FF43"},
	{ChainId: vaa.ChainIDOptimism, Addr: "00000000000000000000000001A957A525a5b7A72808bA9D10c389674E459891"},
	{ChainId: vaa.ChainIDBase, Addr: "000000000000000000000000ea8029CD7FCAEFFcD1F53686430Db0Fc8ed384E1"},
	{ChainId: vaa.ChainIDSeiEVM, Addr: "000000000000000000000000362fca37E45fe1096b42021b543f462D49a5C8df"},
	{ChainId: vaa.ChainIDUnichain, Addr: "000000000000000000000000362fca37E45fe1096b42021b543f462D49a5C8df"},
	{ChainId: vaa.ChainIDInk, Addr: "000000000000000000000000362fca37E45fe1096b42021b543f462D49a5C8df"},
	{ChainId: vaa.ChainIDXRPLEVM, Addr: "000000000000000000000000362fca37E45fe1096b42021b543f462D49a5C8df"},
	{ChainId: vaa.ChainIDSepolia, Addr: "0000000000000000000000007B1bD7a6b4E61c2a123AC6BC2cbfC614437D0470"},
	{ChainId: vaa.ChainIDArbitrumSepolia, Addr: "0000000000000000000000007B1bD7a6b4E61c2a123AC6BC2cbfC614437D0470"},
	{ChainId: vaa.ChainIDOptimismSepolia, Addr: "00000000000000000000000093BAD53DDfB6132b0aC8E37f6029163E63372cEE"},
	{ChainId: vaa.ChainIDBaseSepolia, Addr: "00000000000000000000000093BAD53DDfB6132b0aC8E37f6029163E63372cEE"},
}

// KnownTestnetWrappedNativeAddresses is a list of addresses for deployments of wrapped native asssets (e.g. WETH) on various testnets.
var KnownTestnetWrappedNativeAddresses = map[vaa.ChainID]string{
	// WETH
	vaa.ChainIDSepolia: "0x7b79995e5f793a07bc00c21412e50ecae098e7f9",
	// WETH
	vaa.ChainIDHolesky: "0xc8f93d9738e7Ad5f3aF8c548DB2f6B7F8082B5e8",
}

// KnownTestnetManagerEmitters is a list of known manager emitters on various testnets.
// Note that the format allows multiple emitters per chain.
var KnownTestnetManagerEmitters = []struct {
	ChainId vaa.ChainID
	Addr    string
}{
	{ChainId: vaa.ChainIDSolana, Addr: "af528793be84ee2c922e2b27b7cae282a4d098f4fe528f35ded8c8c06d0b1090"},
}

// TODO: This should be loaded from the delegated manager set contract instead.
// TODO: This should be listed per chain and per index.
// KnownTestnetManagerSet is the initial delegated manager set for testnet.
// It is a 5-of-7 multisig using freshly generated keys.
var KnownTestnetManagerSet = struct {
	M          uint8
	N          uint8
	PublicKeys [][33]byte
}{
	M: 5,
	N: 7,
	PublicKeys: [][33]byte{
		// guardian-0: 02349de56ca5dd06db8660419d6f150662e0f04febdbf6512d7cfe78c23b51491c
		{0x02, 0x34, 0x9D, 0xE5, 0x6C, 0xA5, 0xDD, 0x06, 0xDB, 0x86, 0x60, 0x41, 0x9D, 0x6F, 0x15, 0x06,
			0x62, 0xE0, 0xF0, 0x4F, 0xEB, 0xDB, 0xF6, 0x51, 0x2D, 0x7C, 0xFE, 0x78, 0xC2, 0x3B, 0x51, 0x49,
			0x1C},
		// guardian-1: 035163bfd9518b0a536a17f330a1589fe21d7404b51f525a0a990a65a701952ebb
		{0x03, 0x51, 0x63, 0xBF, 0xD9, 0x51, 0x8B, 0x0A, 0x53, 0x6A, 0x17, 0xF3, 0x30, 0xA1, 0x58, 0x9F,
			0xE2, 0x1D, 0x74, 0x04, 0xB5, 0x1F, 0x52, 0x5A, 0x0A, 0x99, 0x0A, 0x65, 0xA7, 0x01, 0x95, 0x2E,
			0xBB},
		// guardian-2: 036d40b0b85bca49e41f05a26950578bb13a424507ce34a80f83d3cf601e25818b
		{0x03, 0x6D, 0x40, 0xB0, 0xB8, 0x5B, 0xCA, 0x49, 0xE4, 0x1F, 0x05, 0xA2, 0x69, 0x50, 0x57, 0x8B,
			0xB1, 0x3A, 0x42, 0x45, 0x07, 0xCE, 0x34, 0xA8, 0x0F, 0x83, 0xD3, 0xCF, 0x60, 0x1E, 0x25, 0x81,
			0x8B},
		// guardian-3: 0307681002ae28b9399e828d0f46d54c31d5d6ff187b3bdddc6615987a466455f5
		{0x03, 0x07, 0x68, 0x10, 0x02, 0xAE, 0x28, 0xB9, 0x39, 0x9E, 0x82, 0x8D, 0x0F, 0x46, 0xD5, 0x4C,
			0x31, 0xD5, 0xD6, 0xFF, 0x18, 0x7B, 0x3B, 0xDD, 0xDC, 0x66, 0x15, 0x98, 0x7A, 0x46, 0x64, 0x55,
			0xF5},
		// guardian-4: 0375abc8955c8a8c875ee1febd157132adcc1b992d69a946e83485b8360e23a277
		{0x03, 0x75, 0xAB, 0xC8, 0x95, 0x5C, 0x8A, 0x8C, 0x87, 0x5E, 0xE1, 0xFE, 0xBD, 0x15, 0x71, 0x32,
			0xAD, 0xCC, 0x1B, 0x99, 0x2D, 0x69, 0xA9, 0x46, 0xE8, 0x34, 0x85, 0xB8, 0x36, 0x0E, 0x23, 0xA2,
			0x77},
		// guardian-5: 030212d206546216917a75533ed6c975f8f794ba0d8a7fb84dedf65ebb20e64841
		{0x03, 0x02, 0x12, 0xD2, 0x06, 0x54, 0x62, 0x16, 0x91, 0x7A, 0x75, 0x53, 0x3E, 0xD6, 0xC9, 0x75,
			0xF8, 0xF7, 0x94, 0xBA, 0x0D, 0x8A, 0x7F, 0xB8, 0x4D, 0xED, 0xF6, 0x5E, 0xBB, 0x20, 0xE6, 0x48,
			0x41},
		// guardian-6: 037ff483369b52bd87a73f23413dd8fcace71de7f7823c5c9120f1e9cfe5733a88
		{0x03, 0x7F, 0xF4, 0x83, 0x36, 0x9B, 0x52, 0xBD, 0x87, 0xA7, 0x3F, 0x23, 0x41, 0x3D, 0xD8, 0xFC,
			0xAC, 0xE7, 0x1D, 0xE7, 0xF7, 0x82, 0x3C, 0x5C, 0x91, 0x20, 0xF1, 0xE9, 0xCF, 0xE5, 0x73, 0x3A,
			0x88},
	},
}
