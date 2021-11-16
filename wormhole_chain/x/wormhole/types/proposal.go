package types

import (
	"fmt"

	gov "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const (
	ProposalTypeGuardianSetUpdate         string = "GuardianSetUpdate"
	ProposalTypeGovernanceWormholeMessage string = "GovernanceWormholeMessage"
)

func init() {
	gov.RegisterProposalType(ProposalTypeGuardianSetUpdate)
	gov.RegisterProposalTypeCodec(&GuardianSetUpdateProposal{}, "wormhole/GuardianSetUpdate")
	gov.RegisterProposalType(ProposalTypeGovernanceWormholeMessage)
	gov.RegisterProposalTypeCodec(&GovernanceWormholeMessageProposal{}, "wormhole/GovernanceWormholeMessage")
}

func NewGuardianSetUpdateProposal(title, description string, guardianSet GuardianSet) *GuardianSetUpdateProposal {
	return &GuardianSetUpdateProposal{
		Title:          title,
		Description:    description,
		NewGuardianSet: guardianSet,
	}
}

func (sup *GuardianSetUpdateProposal) ProposalRoute() string { return RouterKey }
func (sup *GuardianSetUpdateProposal) ProposalType() string  { return ProposalTypeGuardianSetUpdate }
func (sup *GuardianSetUpdateProposal) ValidateBasic() error {
	if err := sup.NewGuardianSet.ValidateBasic(); err != nil {
		return err
	}
	return gov.ValidateAbstract(sup)
}

func (sup *GuardianSetUpdateProposal) String() string {
	return fmt.Sprintf(`Guardian Set Upgrade Proposal: 
  Title:       %s
  Description: %s
  GuardianSet: %s`, sup.Title, sup.Description, sup.NewGuardianSet.String())
}

func NewGovernanceWormholeMessageProposal(title, description string, action uint8, targetChain uint16, module []byte, payload []byte) *GovernanceWormholeMessageProposal {
	return &GovernanceWormholeMessageProposal{
		Title:       title,
		Description: description,
		Module:      module,
		Action:      uint32(action),
		TargetChain: uint32(targetChain),
		Payload:     payload,
	}
}

func (sup *GovernanceWormholeMessageProposal) ProposalRoute() string { return RouterKey }
func (sup *GovernanceWormholeMessageProposal) ProposalType() string {
	return ProposalTypeGovernanceWormholeMessage
}
func (sup *GovernanceWormholeMessageProposal) ValidateBasic() error {
	if len(sup.Module) != 32 {
		return fmt.Errorf("invalid module length: %d != 32", len(sup.Module))
	}
	return gov.ValidateAbstract(sup)
}

func (sup *GovernanceWormholeMessageProposal) String() string {
	return fmt.Sprintf(`Governance Wormhole Message Proposal: 
  Title:       %s
  Description: %s
  Module: %x
  TargetChain: %d
  Payload: %x`, sup.Title, sup.Description, sup.Module, sup.TargetChain, sup.Payload)
}
