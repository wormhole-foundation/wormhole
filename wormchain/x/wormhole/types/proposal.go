package types

import (
	"fmt"

	govv1beta "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

const (
	ProposalTypeGuardianSetUpdate         string = "GuardianSetUpdate"
	ProposalTypeGovernanceWormholeMessage string = "GovernanceWormholeMessage"
)

func init() {
	govv1beta.RegisterProposalType(ProposalTypeGuardianSetUpdate)
	govv1beta.RegisterProposalType(ProposalTypeGovernanceWormholeMessage)
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
	return govv1beta.ValidateAbstract(sup)
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
	return govv1beta.ValidateAbstract(sup)
}

func (sup *GovernanceWormholeMessageProposal) String() string {
	return fmt.Sprintf(`Governance Wormhole Message Proposal: 
  Title:       %s
  Description: %s
  Module: %x
  TargetChain: %d
  Payload: %x`, sup.Title, sup.Description, sup.Module, sup.TargetChain, sup.Payload)
}
