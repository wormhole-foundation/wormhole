package sdk

import (
	"encoding/hex"
	"fmt"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// PublicRPCEndpoints is a list of known public RPC endpoints for mainnet, operated by
// Wormhole guardian nodes.
//
// This list is duplicated a couple times across the codebase - make sure to update all copies!
var PublicRPCEndpoints = []string{
	"https://wormhole-v2-mainnet-api.mcf.rocks",
	"https://wormhole-v2-mainnet-api.chainlayer.network",
	"https://wormhole-v2-mainnet-api.staking.fund",
	"https://guardian.mainnet.xlabs.xyz",
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

type EmitterInfo struct {
	ChainID    vaa.ChainID
	Emitter    string
	BridgeType EmitterType
}

// KnownEmitters is a list of well-known mainnet emitters we want to take into account
// when iterating over all emitters - like for finding and repairing missing messages.
//
// Wormhole is not permissioned - anyone can use it. Adding contracts to this list is
// entirely optional and at the core team's discretion.
var KnownEmitters = buildKnownEmitters(knownTokenbridgeEmitters, knownNFTBridgeEmitters)

func buildKnownEmitters(tokenEmitters, nftEmitters map[vaa.ChainID]string) []EmitterInfo {
	out := make([]EmitterInfo, 0, len(knownTokenbridgeEmitters)+len(knownNFTBridgeEmitters))
	for id, emitter := range tokenEmitters {
		out = append(out, EmitterInfo{
			ChainID:    id,
			Emitter:    emitter,
			BridgeType: EmitterTokenBridge,
		})
	}

	for id, emitter := range nftEmitters {
		out = append(out, EmitterInfo{
			ChainID:    id,
			Emitter:    emitter,
			BridgeType: EmitterNFTBridge,
		})
	}

	return out
}

func buildEmitterMap(hexmap map[vaa.ChainID]string) map[vaa.ChainID][]byte {
	out := make(map[vaa.ChainID][]byte)
	for id, emitter := range hexmap {
		e, err := hex.DecodeString(emitter)
		if err != nil {
			panic(fmt.Sprintf("Failed to decode emitter address %v: %v", emitter, err))
		}
		out[id] = e
	}

	return out
}

// KnownTokenbridgeEmitters is a list of well-known mainnet emitters for the tokenbridge.
var KnownTokenbridgeEmitters = buildEmitterMap(knownTokenbridgeEmitters)
var knownTokenbridgeEmitters = map[vaa.ChainID]string{
	vaa.ChainIDSolana:     "ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5",
	vaa.ChainIDEthereum:   "0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585",
	vaa.ChainIDTerra:      "0000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2",
	vaa.ChainIDTerra2:     "a463ad028fb79679cfc8ce1efba35ac0e77b35080a1abe9bebe83461f176b0a3",
	vaa.ChainIDBSC:        "000000000000000000000000b6f6d86a8f9879a9c87f643768d9efc38c1da6e7",
	vaa.ChainIDPolygon:    "0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde",
	vaa.ChainIDAvalanche:  "0000000000000000000000000e082f06ff657d94310cb8ce8b0d9a04541d8052",
	vaa.ChainIDOasis:      "0000000000000000000000005848c791e09901b40a9ef749f2a6735b418d7564",
	vaa.ChainIDAlgorand:   "67e93fa6c8ac5c819990aa7340c0c16b508abb1178be9b30d024b8ac25193d45",
	vaa.ChainIDAptos:      "0000000000000000000000000000000000000000000000000000000000000001",
	vaa.ChainIDAurora:     "00000000000000000000000051b5123a7b0F9b2bA265f9c4C8de7D78D52f510F",
	vaa.ChainIDFantom:     "0000000000000000000000007C9Fc5741288cDFdD83CeB07f3ea7e22618D79D2",
	vaa.ChainIDKarura:     "000000000000000000000000ae9d7fe007b3327AA64A32824Aaac52C42a6E624",
	vaa.ChainIDAcala:      "000000000000000000000000ae9d7fe007b3327AA64A32824Aaac52C42a6E624",
	vaa.ChainIDKlaytn:     "0000000000000000000000005b08ac39EAED75c0439FC750d9FE7E1F9dD0193F",
	vaa.ChainIDCelo:       "000000000000000000000000796Dff6D74F3E27060B71255Fe517BFb23C93eed",
	vaa.ChainIDNear:       "148410499d3fcda4dcfd68a1ebfcdddda16ab28326448d4aae4d2f0465cdfcb7",
	vaa.ChainIDMoonbeam:   "000000000000000000000000B1731c586ca89a23809861c6103F0b96B3F57D92",
	vaa.ChainIDArbitrum:   "0000000000000000000000000b2402144Bb366A632D14B83F244D2e0e21bD39c",
	vaa.ChainIDOptimism:   "0000000000000000000000001D68124e65faFC907325e3EDbF8c4d84499DAa8b",
	vaa.ChainIDBase:       "0000000000000000000000008d2de8d2f73F1F4cAB472AC9A881C9b123C79627",
	vaa.ChainIDXpla:       "8f9cf727175353b17a5f574270e370776123d90fd74956ae4277962b4fdee24c",
	vaa.ChainIDScroll:     "00000000000000000000000024850c6f61C438823F01B7A3BF2B89B72174Fa9d",
	vaa.ChainIDMantle:     "00000000000000000000000024850c6f61C438823F01B7A3BF2B89B72174Fa9d",
	vaa.ChainIDBlast:      "00000000000000000000000024850c6f61C438823F01B7A3BF2B89B72174Fa9d",
	vaa.ChainIDXLayer:     "0000000000000000000000005537857664B0f9eFe38C9f320F75fEf23234D904",
	vaa.ChainIDBerachain:  "0000000000000000000000003Ff72741fd67D6AD0668d93B41a09248F4700560",
	vaa.ChainIDSeiEVM:     "0000000000000000000000003Ff72741fd67D6AD0668d93B41a09248F4700560",
	vaa.ChainIDSnaxchain:  "0000000000000000000000008B94bfE456B48a6025b92E11Be393BAa86e68410",
	vaa.ChainIDUnichain:   "0000000000000000000000003Ff72741fd67D6AD0668d93B41a09248F4700560",
	vaa.ChainIDInjective:  "00000000000000000000000045dbea4617971d93188eda21530bc6503d153313",
	vaa.ChainIDSui:        "ccceeb29348f71bdd22ffef43a2a19c1f5b5e17c5cca5411529120182672ade5",
	vaa.ChainIDSei:        "86c5fd957e2db8389553e1728f9c27964b22a8154091ccba54d75f4b10c61f5e",
	vaa.ChainIDWormchain:  "aeb534c45c3049d380b9d9b966f9895f53abd4301bfaff407fa09dea8ae7a924",
	vaa.ChainIDWorldchain: "000000000000000000000000c309275443519adca74c9136b02A38eF96E3a1f6",
	vaa.ChainIDInk:        "0000000000000000000000003Ff72741fd67D6AD0668d93B41a09248F4700560",
}

// KnownNFTBridgeEmitters is a list of well-known mainnet emitters for the NFT bridge.
var KnownNFTBridgeEmitters = buildEmitterMap(knownNFTBridgeEmitters)
var knownNFTBridgeEmitters = map[vaa.ChainID]string{
	vaa.ChainIDSolana:    "0def15a24423e1edd1a5ab16f557b9060303ddbab8c803d2ee48f4b78a1cfd6b",
	vaa.ChainIDEthereum:  "0000000000000000000000006ffd7ede62328b3af38fcd61461bbfc52f5651fe",
	vaa.ChainIDBSC:       "0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde",
	vaa.ChainIDPolygon:   "00000000000000000000000090bbd86a6fe93d3bc3ed6335935447e75fab7fcf",
	vaa.ChainIDAvalanche: "000000000000000000000000f7b6737ca9c4e08ae573f75a97b73d7a813f5de5",
	vaa.ChainIDOasis:     "00000000000000000000000004952d522ff217f40b5ef3cbf659eca7b952a6c1",
	vaa.ChainIDAurora:    "0000000000000000000000006dcC0484472523ed9Cdc017F711Bcbf909789284",
	vaa.ChainIDFantom:    "000000000000000000000000A9c7119aBDa80d4a4E0C06C8F4d8cF5893234535",
	vaa.ChainIDKarura:    "000000000000000000000000b91e3638F82A1fACb28690b37e3aAE45d2c33808",
	vaa.ChainIDAcala:     "000000000000000000000000b91e3638F82A1fACb28690b37e3aAE45d2c33808",
	vaa.ChainIDKlaytn:    "0000000000000000000000003c3c561757BAa0b78c5C025CdEAa4ee24C1dFfEf",
	vaa.ChainIDCelo:      "000000000000000000000000A6A377d75ca5c9052c9a77ED1e865Cc25Bd97bf3",
	vaa.ChainIDMoonbeam:  "000000000000000000000000453cfBe096C0f8D763E8C5F24B441097d577bdE2",
	vaa.ChainIDArbitrum:  "0000000000000000000000003dD14D553cFD986EAC8e3bddF629d82073e188c8",
	vaa.ChainIDOptimism:  "000000000000000000000000fE8cD454b4A1CA468B57D79c0cc77Ef5B6f64585",
	vaa.ChainIDBase:      "000000000000000000000000DA3adC6621B2677BEf9aD26598e6939CF0D92f88",
	vaa.ChainIDAptos:     "0000000000000000000000000000000000000000000000000000000000000005",
}

func GetEmitterAddressForChain(chainID vaa.ChainID, emitterType EmitterType) (vaa.Address, error) {
	for _, emitter := range KnownEmitters {
		if emitter.ChainID == chainID && emitter.BridgeType == emitterType {
			emitterAddr, err := vaa.StringToAddress(emitter.Emitter)
			if err != nil {
				return vaa.Address{}, err
			}

			return emitterAddr, nil
		}
	}

	return vaa.Address{}, fmt.Errorf("lookup failed")
}

// KnownAutomaticRelayerEmitters is a list of well-known mainnet emitters for the Automatic Relayers.
// It is based on this: https://github.com/wormhole-foundation/wormhole/blob/2c9703670eadc48a7dc8967e81ed2823affcc679/sdk/js/src/relayer/consts.ts#L95
// Note that the format of this is different from the other maps because we don't want to limit it to one per chain.
var KnownAutomaticRelayerEmitters = []struct {
	ChainId vaa.ChainID
	Addr    string
}{
	{ChainId: vaa.ChainIDEthereum, Addr: "00000000000000000000000027428DD2d3DD32A4D7f7C497eAaa23130d894911"},
	{ChainId: vaa.ChainIDBSC, Addr: "00000000000000000000000027428DD2d3DD32A4D7f7C497eAaa23130d894911"},
	{ChainId: vaa.ChainIDPolygon, Addr: "00000000000000000000000027428DD2d3DD32A4D7f7C497eAaa23130d894911"},
	{ChainId: vaa.ChainIDAvalanche, Addr: "00000000000000000000000027428DD2d3DD32A4D7f7C497eAaa23130d894911"},
	{ChainId: vaa.ChainIDFantom, Addr: "00000000000000000000000027428DD2d3DD32A4D7f7C497eAaa23130d894911"},
	{ChainId: vaa.ChainIDKlaytn, Addr: "00000000000000000000000027428DD2d3DD32A4D7f7C497eAaa23130d894911"},
	{ChainId: vaa.ChainIDCelo, Addr: "00000000000000000000000027428DD2d3DD32A4D7f7C497eAaa23130d894911"},
	{ChainId: vaa.ChainIDAcala, Addr: "00000000000000000000000027428DD2d3DD32A4D7f7C497eAaa23130d894911"},
	{ChainId: vaa.ChainIDKarura, Addr: "00000000000000000000000027428DD2d3DD32A4D7f7C497eAaa23130d894911"},
	{ChainId: vaa.ChainIDMoonbeam, Addr: "00000000000000000000000027428DD2d3DD32A4D7f7C497eAaa23130d894911"},
	{ChainId: vaa.ChainIDArbitrum, Addr: "00000000000000000000000027428DD2d3DD32A4D7f7C497eAaa23130d894911"},
	{ChainId: vaa.ChainIDOptimism, Addr: "00000000000000000000000027428DD2d3DD32A4D7f7C497eAaa23130d894911"},
	{ChainId: vaa.ChainIDBase, Addr: "000000000000000000000000706f82e9bb5b0813501714ab5974216704980e31"},
	{ChainId: vaa.ChainIDScroll, Addr: "00000000000000000000000027428DD2d3DD32A4D7f7C497eAaa23130d894911"},
	{ChainId: vaa.ChainIDBlast, Addr: "00000000000000000000000027428DD2d3DD32A4D7f7C497eAaa23130d894911"},
	{ChainId: vaa.ChainIDMantle, Addr: "00000000000000000000000027428dd2d3dd32a4d7f7c497eaaa23130d894911"},
	{ChainId: vaa.ChainIDXLayer, Addr: "00000000000000000000000027428dd2d3dd32a4d7f7c497eaaa23130d894911"},
	{ChainId: vaa.ChainIDSnaxchain, Addr: "00000000000000000000000027428DD2d3DD32A4D7f7C497eAaa23130d894911"},
	{ChainId: vaa.ChainIDWorldchain, Addr: "0000000000000000000000001520cc9e779c56dab5866bebfb885c86840c33d3"},
}

// KnownWrappedNativeAddress is a map of wrapped native addresses by chain ID, e.g. WETH for Ethereum
var KnownWrappedNativeAddress = map[vaa.ChainID]string{
	// WETH
	vaa.ChainIDEthereum: "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
}
