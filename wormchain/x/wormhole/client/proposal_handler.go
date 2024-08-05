package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/wormhole-foundation/wormchain/x/wormhole/client/cli"
)

var GuardianSetUpdateProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitGuardianSetUpdateProposal)
var WormholeGovernanceMessageProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitWormholeGovernanceMessageProposal)
