package cli

import (
	"fmt"
	// "strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	// sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(queryRoute string) *cobra.Command {
	// Group wormhole queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdListGuardianSet())
	cmd.AddCommand(CmdShowGuardianSet())
	cmd.AddCommand(CmdShowConfig())
	cmd.AddCommand(CmdListReplayProtection())
	cmd.AddCommand(CmdShowReplayProtection())
	cmd.AddCommand(CmdListSequenceCounter())
	cmd.AddCommand(CmdShowSequenceCounter())
	cmd.AddCommand(CmdShowConsensusGuardianSetIndex())
	cmd.AddCommand(CmdListGuardianValidator())
	cmd.AddCommand(CmdShowGuardianValidator())
	cmd.AddCommand(CmdLatestGuardianSetIndex())
	cmd.AddCommand(CmdListAllowlists())
	cmd.AddCommand(CmdShowAllowlist())
	cmd.AddCommand(CmdShowIbcComposabilityMwContract())
	cmd.AddCommand(CmdListWasmInstantiateAllowlist())

	// this line is used by starport scaffolding # 1

	return cmd
}
