package txverifier

import "github.com/wormhole-foundation/wormhole/sdk/vaa"

func ParameterRange(input []uint) []vaa.ChainID {
	result := make([]vaa.ChainID, 0, len(input))
	for _, chain := range input {
		result = append(result, vaa.ChainID(chain))
	}
	return result
}

func LocalRangeNearMiss() []vaa.ChainID {
	input := []uint{1, 2}
	result := make([]vaa.ChainID, 0, len(input))
	for _, chain := range input {
		result = append(result, vaa.ChainID(chain))
	}
	return result
}
