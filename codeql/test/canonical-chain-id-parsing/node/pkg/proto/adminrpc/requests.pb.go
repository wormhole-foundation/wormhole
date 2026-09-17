package adminrpc

type ReobserveRequest struct {
	ChainId uint32
}

func (r *ReobserveRequest) GetChainId() uint32 {
	if r == nil {
		return 0
	}
	return r.ChainId
}

type GovernanceRequest struct {
	ChainId    uint32
	NewChainId uint32
	Version    uint32
}

func (r *GovernanceRequest) GetChainId() uint32 {
	if r == nil {
		return 0
	}
	return r.ChainId
}

func (r *GovernanceRequest) GetNewChainId() uint32 {
	if r == nil {
		return 0
	}
	return r.NewChainId
}
