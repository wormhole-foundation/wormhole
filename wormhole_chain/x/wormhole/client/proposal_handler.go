package client

import (
	"github.com/certusone/wormhole-chain/x/wormhole/client/cli"
	"github.com/certusone/wormhole-chain/x/wormhole/client/rest"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

var GuardianSetUpdateProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitGuardianSetUpdateProposal, rest.ProposalGuardianSetUpdateRESTHandler)
var WormholeGovernanceMessageProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitWormholeGovernanceMessageProposal, rest.ProposalWormholeGovernanceMessageRESTHandler)
