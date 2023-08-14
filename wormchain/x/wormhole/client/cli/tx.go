package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
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
	cmd.AddCommand(CmdMigrateContract())
	cmd.AddCommand(CmdCreateAllowedAddress())
	cmd.AddCommand(CmdDeleteAllowedAddress())
	cmd.AddCommand(CmdAddWasmInstantiateAllowlist())
	cmd.AddCommand(CmdDeleteWasmInstantiateAllowlist())
	cmd.AddCommand(CmdExecuteGatewayGovernanceVaa())
	// this line is used by starport scaffolding # 1

	return cmd
}
