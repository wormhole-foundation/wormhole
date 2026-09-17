package adminrpc

import (
	"strconv"

	proto "github.com/wormhole-foundation/wormhole/node/pkg/proto/adminrpc"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func CanonicalWireValidHelper(req *proto.GovernanceRequest) (BodyContractUpgrade, error) {
	chain, err := vaa.ChainIDFromNumber[uint32](req.ChainId)
	if err != nil {
		return BodyContractUpgrade{}, err
	}
	return BodyContractUpgrade{TargetChainID: chain}, nil
}

func CanonicalKnownHelper(req *proto.ReobserveRequest, reobservers map[vaa.ChainID]string) (string, error) {
	chain, err := vaa.KnownChainIDFromNumber[uint32](req.GetChainId())
	if err != nil {
		return "", err
	}
	return reobservers[chain], nil
}

func CanonicalStringKnownHelper(chainIDStr string, out map[vaa.ChainID]bool) error {
	chain, err := vaa.StringToKnownChainID(chainIDStr)
	if err != nil {
		return err
	}
	out[chain] = true
	return nil
}

func InternalTypedValue(chain vaa.ChainID) BodyContractUpgrade {
	return BodyContractUpgrade{TargetChainID: chain}
}

func InternalTypedConstant() BodyContractUpgrade {
	return BodyContractUpgrade{TargetChainID: vaa.ChainIDPythNet}
}

func NonChainIDVersion(req *proto.GovernanceRequest) uint16 {
	if req.Version > 255 {
		return 0
	}
	return uint16(req.Version)
}

func GovernanceVersionSkewUsesWireValidHelper(req *proto.GovernanceRequest) (BodyContractUpgrade, error) {
	chain, err := vaa.ChainIDFromNumber[uint32](req.GetChainId())
	if err != nil {
		return BodyContractUpgrade{}, err
	}
	return BodyContractUpgrade{TargetChainID: chain}, nil
}

func WireValidStringException(chainIDStr string) (vaa.ChainID, error) {
	parsed, err := strconv.ParseUint(chainIDStr, 10, 16)
	if err != nil {
		return 0, err
	}
	return vaa.ChainIDFromNumber[uint64](parsed)
}

type Attributes struct{}

func (Attributes) GetAsUint(_ string, _ int) (uint64, error) {
	return 0, nil
}

func SameNamedIbcAttributeOutsideIbcWatcher(attributes Attributes) BodyContractUpgrade {
	chain, err := attributes.GetAsUint("message.chain_id", 16)
	if err != nil {
		return BodyContractUpgrade{}
	}
	return BodyContractUpgrade{TargetChainID: vaa.ChainID(chain)}
}

func JsonFloat64OutsideIbcWatcher(entry []any) vaa.ChainID {
	chain, ok := entry[1].(float64)
	if !ok {
		return 0
	}
	return vaa.ChainID(chain)
}

func ParameterRangeOutsideTxVerifier(input []uint) []vaa.ChainID {
	result := make([]vaa.ChainID, 0, len(input))
	for _, chain := range input {
		result = append(result, vaa.ChainID(chain))
	}
	return result
}
