package types

import (
	"github.com/certusone/wormhole-chain/x/wormhole/client/cli"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
)

// TODO add rest because this will crash on launch
var GuardianSetUpdateProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitGuardianSetUpdateProposal, nil)
var WormholeGovernanceMessageProposalHandler = govclient.NewProposalHandler(cli.NewCmdSubmitWormholeGovernanceMessageProposal, nil)
