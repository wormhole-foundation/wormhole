package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/wormhole-foundation/wormhole-chain/x/wormhole/types"
)

// GetTxCmd returns the transaction commands for this module
func GetTxCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      fmt.Sprintf("%s transactions subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdExecuteGovernanceVAA())
	cmd.AddCommand(CmdRegisterAccountAsGuardian())
	cmd.AddCommand(CmdStoreCode())
	cmd.AddCommand(CmdInstantiateContract())
	// this line is used by starport scaffolding # 1

	return cmd
}
