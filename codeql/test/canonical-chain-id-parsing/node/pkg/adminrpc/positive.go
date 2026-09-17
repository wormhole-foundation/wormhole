package adminrpc

import (
	"math"

	proto "github.com/wormhole-foundation/wormhole/node/pkg/proto/adminrpc"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type BodyContractUpgrade struct {
	TargetChainID vaa.ChainID
}

type BodyGuardianSetUpdate struct {
	OldChain vaa.ChainID
	NewChain vaa.ChainID
}

func submitGovernanceBody(body BodyContractUpgrade) {}

func useKnownChain(chain vaa.ChainID) {}

func DirectFieldCastWireValid(req *proto.GovernanceRequest) {
	if req.ChainId > math.MaxUint16 {
		return
	}
	submitGovernanceBody(BodyContractUpgrade{TargetChainID: vaa.ChainID(req.ChainId)})
}

func DirectGetterCastKnown(req *proto.ReobserveRequest, reobservers map[vaa.ChainID]string) string {
	if req.GetChainId() > math.MaxUint16 {
		return ""
	}
	return reobservers[vaa.ChainID(req.GetChainId())]
}

func AliasThenCallArgument(req *proto.ReobserveRequest) {
	alias := req
	chainNumber := alias.GetChainId()
	if chainNumber > math.MaxUint16 {
		return
	}
	useKnownChain(vaa.ChainID(chainNumber))
}

func ChannelReceiveBoundary(ch <-chan *proto.ReobserveRequest, expected vaa.ChainID) bool {
	req := <-ch
	if req.ChainId > math.MaxUint16 {
		return false
	}
	return vaa.ChainID(req.ChainId) == expected
}

func MultipleGeneratedFields(req *proto.GovernanceRequest) BodyGuardianSetUpdate {
	if req.ChainId > math.MaxUint16 || req.NewChainId > math.MaxUint16 {
		return BodyGuardianSetUpdate{}
	}
	return BodyGuardianSetUpdate{
		OldChain: vaa.ChainID(req.ChainId),
		NewChain: vaa.ChainID(req.NewChainId),
	}
}
