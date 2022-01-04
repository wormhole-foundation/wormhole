package cli

import (
	"fmt"
	// "strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	// sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
)

// GetQueryCmd returns the cli query commands for this module
func GetQueryCmd(queryRoute string) *cobra.Command {
	// Group tokenbridge queries under a subcommand
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("Querying commands for the %s module", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdShowConfig())
	cmd.AddCommand(CmdListReplayProtection())
	cmd.AddCommand(CmdShowReplayProtection())
	cmd.AddCommand(CmdListChainRegistration())
	cmd.AddCommand(CmdShowChainRegistration())
	cmd.AddCommand(CmdListCoinMetaRollbackProtection())
	cmd.AddCommand(CmdShowCoinMetaRollbackProtection())
	// this line is used by starport scaffolding # 1

	return cmd
}
