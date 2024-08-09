package helpers

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type TbInstantiateMsg struct {
	GovChain   uint16 `json:"gov_chain"`
	GovAddress []byte `json:"gov_address"`

	WormholeContract   string `json:"wormhole_contract"`
	WrappedAssetCodeId uint64 `json:"wrapped_asset_code_id"`

	ChainId        uint16 `json:"chain_id"`
	NativeDenom    string `json:"native_denom"`
	NativeSymbol   string `json:"native_symbol"`
	NativeDecimals uint8  `json:"native_decimals"`
}

func TbContractInstantiateMsg(t *testing.T, cfg ibc.ChainConfig, whContract string, wrappedAssetCodeId string) string {
	codeId, err := strconv.ParseUint(wrappedAssetCodeId, 10, 64)
	require.NoError(t, err)

	msg := TbInstantiateMsg{
		GovChain:           uint16(vaa.GovernanceChain),
		GovAddress:         vaa.GovernanceEmitter[:],
		WormholeContract:   whContract,
		WrappedAssetCodeId: codeId,
		ChainId:            uint16(vaa.ChainIDWormchain),
		NativeDenom:        cfg.Denom,
		NativeSymbol:       "WORM",
		NativeDecimals:     6,
	}
	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return string(msgBz)
}

type TbSubmitVaaMsg struct {
	SubmitVaa SubmitVaa `json:"submit_vaa,omitempty"`
}

type SubmitVaa struct {
	Data []byte `json:"data,omitempty"`
}

func TbRegisterChainMsg(t *testing.T, chainID uint16, emitterAddr string, guardians *guardians.ValSet) []byte {
	emitterBz := [32]byte{}
	eIndex := 32
	for i := len(emitterAddr); i > 0; i-- {
		emitterBz[eIndex-1] = emitterAddr[i-1]
		eIndex--
	}
	bodyTbRegisterChain := vaa.BodyTokenBridgeRegisterChain{
		Module:         "TokenBridge",
		ChainID:        vaa.ChainID(chainID),
		EmitterAddress: vaa.Address(emitterBz),
	}

	payload, err := bodyTbRegisterChain.Serialize()
	require.NoError(t, err)
	v := generateVaa(0, guardians, vaa.ChainID(vaa.GovernanceChain), vaa.Address(vaa.GovernanceEmitter), payload)
	vBz, err := v.Marshal()
	require.NoError(t, err)

	msg := TbSubmitVaaMsg{
		SubmitVaa: SubmitVaa{
			Data: vBz,
		},
	}

	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return msgBz
}

func TbRegisterForeignAsset(t *testing.T, tokenAddr string, chainID uint16, emitterAddr string, decimals uint8, symbol string, name string, guardians *guardians.ValSet) []byte {
	payload := new(bytes.Buffer)
	vaa.MustWrite(payload, binary.BigEndian, uint8(2))
	tokenAddrPadded, err := vaa.LeftPadBytes(tokenAddr, 32)
	require.NoError(t, err)
	payload.Write(tokenAddrPadded.Bytes())
	vaa.MustWrite(payload, binary.BigEndian, chainID)
	vaa.MustWrite(payload, binary.BigEndian, decimals)
	symbolPad := make([]byte, 32)
	copy(symbolPad, []byte(symbol))
	payload.Write(symbolPad)
	namePad := make([]byte, 32)
	copy(namePad, []byte(name))
	payload.Write(namePad)

	emitterBz := [32]byte{}
	eIndex := 32
	for i := len(emitterAddr); i > 0; i-- {
		emitterBz[eIndex-1] = emitterAddr[i-1]
		eIndex--
	}
	v := generateVaa(0, guardians, vaa.ChainID(chainID), vaa.Address(emitterBz), payload.Bytes())
	vBz, err := v.Marshal()
	require.NoError(t, err)

	msg := TbSubmitVaaMsg{
		SubmitVaa: SubmitVaa{
			Data: vBz,
		},
	}

	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return msgBz
}

type TbQueryMsg struct {
	WrappedRegistry WrappedRegistry `json:"wrapped_registry"`
}

type WrappedRegistry struct {
	Chain   uint16 `json:"chain"`
	Address []byte `json:"address"`
}

func CreateCW20Query(t *testing.T, chainID uint16, address string) TbQueryMsg {
	addressBz, err := vaa.LeftPadBytes(address, 32)
	require.NoError(t, err)
	msg := TbQueryMsg{
		WrappedRegistry: WrappedRegistry{
			Chain:   chainID,
			Address: addressBz.Bytes(),
		},
	}
	return msg
}

type TbQueryRsp struct {
	Data *TbQueryRspObj `json:"data,omitempty"`
}

type TbQueryRspObj struct {
	Address string `json:"address"`
}
