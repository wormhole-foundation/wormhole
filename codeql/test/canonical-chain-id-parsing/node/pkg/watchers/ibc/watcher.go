package ibc

import "github.com/wormhole-foundation/wormhole/sdk/vaa"

type MessagePublication struct {
	EmitterChain vaa.ChainID
}

type WasmAttributes struct{}

func (*WasmAttributes) GetAsUint(_ string, _ int) (uint64, error) {
	return 0, nil
}

type OtherAttributes struct{}

func (*OtherAttributes) GetAsUint(_ string, _ int) (uint64, error) {
	return 0, nil
}

func IbcMessageChainID(attributes *WasmAttributes) MessagePublication {
	chain, err := attributes.GetAsUint("message.chain_id", 16)
	if err != nil {
		return MessagePublication{}
	}
	return MessagePublication{EmitterChain: vaa.ChainID(chain)}
}

type queryResults struct {
	Data struct {
		ChannelChains [][]any
		OtherChains   [][]any
	}
}

func JsonChannelMapping(result queryResults) map[string]vaa.ChainID {
	ret := make(map[string]vaa.ChainID)
	for _, entry := range result.Data.ChannelChains {
		channelID, ok := entry[0].(string)
		if !ok {
			return ret
		}
		chain, ok := entry[1].(float64)
		if !ok {
			return ret
		}
		chainID := vaa.ChainID(chain)
		ret[channelID] = chainID
	}
	return ret
}

func IbcNearMissWrongReceiver(attributes *OtherAttributes) MessagePublication {
	chain, err := attributes.GetAsUint("message.chain_id", 16)
	if err != nil {
		return MessagePublication{}
	}
	return MessagePublication{EmitterChain: vaa.ChainID(chain)}
}

func IbcNearMissWrongAttribute(attributes *WasmAttributes) MessagePublication {
	chain, err := attributes.GetAsUint("message.nonce", 16)
	if err != nil {
		return MessagePublication{}
	}
	return MessagePublication{EmitterChain: vaa.ChainID(chain)}
}

func IbcNearMissWrongBitSize(attributes *WasmAttributes) MessagePublication {
	chain, err := attributes.GetAsUint("message.chain_id", 32)
	if err != nil {
		return MessagePublication{}
	}
	return MessagePublication{EmitterChain: vaa.ChainID(chain)}
}

func JsonNearMissWrongIndex(result queryResults) vaa.ChainID {
	for _, entry := range result.Data.ChannelChains {
		chain, ok := entry[0].(float64)
		if !ok {
			return 0
		}
		return vaa.ChainID(chain)
	}
	return 0
}

func JsonNearMissWrongDataShape(result queryResults) vaa.ChainID {
	for _, entry := range result.Data.OtherChains {
		chain, ok := entry[1].(float64)
		if !ok {
			return 0
		}
		return vaa.ChainID(chain)
	}
	return 0
}

func JsonNearMissDirectParameter(entry []any) vaa.ChainID {
	chain, ok := entry[0].(float64)
	if !ok {
		return 0
	}
	return vaa.ChainID(chain)
}
